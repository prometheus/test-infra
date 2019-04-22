package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/pkg/provider/k8s"
	"gopkg.in/alecthomas/kingpin.v2"
	podV1 "k8s.io/kubernetes/pkg/api/v1/pod"
)

type operator struct {
	context  context.Context
	cancel   context.CancelFunc
	blocker  chan struct{}
	readyErr []error
	execErr  []error
	wg       sync.WaitGroup
}

type restart struct {
	k8sClient *k8s.K8s
}

func newRestart() *restart {
	k, err := k8s.New(context.Background(), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error creating k8s client inside the k8s cluster"))
		os.Exit(2)
	}
	return &restart{
		k8sClient: k,
	}
}

func newOperator() *operator {
	var o operator
	o.blocker = make(chan struct{})
	o.context, o.cancel = context.WithCancel(context.Background())
	return &o
}

func (s *restart) killPrometheus(o *operator, ready chan struct{}, podName, namespace string) {
	defer o.wg.Done()
	command := "/scripts/killer.sh"
	container := "prometheus"

	// fetch latest pod status
	podList, _ := s.k8sClient.FetchRunningPods(namespace, "", "metadata.name="+podName)
	pod := podList.Items[0]

	rPS := pod.Spec.Containers[0].ReadinessProbe.PeriodSeconds
	rFT := pod.Spec.Containers[0].ReadinessProbe.FailureThreshold

	if !podV1.IsPodReady(&pod) {
		o.readyErr = append(o.readyErr, fmt.Errorf("prometheus container not ready for %v, won't kill", podName))
	}
	ready <- struct{}{}

	log.Printf("blocking killPrometheus for %v", podName)
	select {
	case <-o.context.Done():
		log.Printf("cancelling killPrometeus for %v", podName)
		return
	case <-o.blocker:
		_, err := s.k8sClient.ExecuteInPod(command, podName, container, namespace)
		if err != nil {
			o.execErr = append(o.execErr, errors.Wrapf(err, "killPrometheus failed for %v", podName))
			return
		}
		// sleep till k8s marks the pod as not ready
		log.Printf("killPrometheus succcessful for %v : waiting: %vs", podName, rFT*rPS+rPS)
		time.Sleep(time.Duration(rFT*rPS+rPS) * time.Second)
		return
	}

}

func (s *restart) startPrometheus(o *operator, ready chan struct{}, podName, namespace string) {
	defer o.wg.Done()
	command := "/scripts/restarter.sh"
	container := "prometheus"

	// fetch latest pod status
	podList, _ := s.k8sClient.FetchRunningPods(namespace, "", "metadata.name="+podName)
	pod := podList.Items[0]

	rPS := pod.Spec.Containers[0].ReadinessProbe.PeriodSeconds
	rST := pod.Spec.Containers[0].ReadinessProbe.SuccessThreshold

	if podV1.IsPodReady(&pod) {
		o.readyErr = append(o.readyErr, fmt.Errorf("prometheus container is already ready for %v, won't start", podName))
	}
	ready <- struct{}{}

	log.Printf("blocking startPrometheus for %v", podName)
	select {
	case <-o.context.Done():
		log.Printf("cancelling startPrometeus for %v", podName)
		return
	case <-o.blocker:
		_, err := s.k8sClient.ExecuteInPod(command, podName, container, namespace)
		if err != nil {
			o.execErr = append(o.execErr, errors.Wrapf(err, "startPrometheus failed for %v", podName))
			return
		}
		// sleep till k8s marks the pod as ready
		log.Printf("startPrometheus succcessful for %v : waiting: %vs", podName, rST*rPS+rPS)
		time.Sleep(time.Duration(rST*rPS+rPS) * time.Second)
		return
	}
}

func (s *restart) restart(*kingpin.ParseContext) error {
	log.Printf("Starting Prombench-Restarter")

	prNo := s.k8sClient.DeploymentVars["PR_NUMBER"]
	namespace := "prombench-" + prNo
	ready := make(chan struct{})

	for {
		log.Println("***** restarting prometheus ******")

		killer := newOperator()
		starter := newOperator()

		// get podList everytime because podName can change
		podList, err := s.k8sClient.FetchRunningPods(namespace, "app=prometheus", "")
		if err != nil {
			log.Fatalf("error fetching pods, restarting restarter: %v", err)
		}
		// for we run 2 prometheus instances currently
		if len(podList.Items) != 2 {
			log.Fatalln("all pods not returned, restarting restarter")
		}
		// restart restarter if all pods are not in the same state
		prevPodStatus := podV1.IsPodReady(&podList.Items[0])
		for _, pod := range podList.Items[1:] {
			if podV1.IsPodReady(&pod) != prevPodStatus {
				log.Fatalln("all pods do not have the same status, restarting restarter")
			}
			prevPodStatus = podV1.IsPodReady(&pod)
		}

		// kill prometheus
		for _, pod := range podList.Items {
			killer.wg.Add(1)
			go s.killPrometheus(killer, ready, pod.ObjectMeta.Name, namespace)
			<-ready
		}
		if len(killer.readyErr) != 0 {
			for _, e := range killer.readyErr {
				log.Println(e)
			}
			killer.cancel()
		} else {
			close(killer.blocker)
		}
		killer.wg.Wait()
		if len(killer.execErr) != 0 {
			for _, e := range killer.execErr {
				log.Println(e)
			}
			// maybe should add a metric here for alerts
			// failing here means it was not able to kill one or more prometheus
		}

		// start prometheus
		for _, pod := range podList.Items {
			starter.wg.Add(1)
			go s.startPrometheus(starter, ready, pod.ObjectMeta.Name, namespace)
			<-ready
		}
		if len(starter.readyErr) != 0 {
			for _, e := range starter.readyErr {
				log.Println(e)
			}
			starter.cancel()
		} else {
			close(starter.blocker)
		}
		starter.wg.Wait()
		if len(starter.execErr) != 0 {
			for _, e := range starter.execErr {
				log.Println(e)
			}
			// maybe should add a metric here for alerts
			// failing here means it was not able to start one or more prometheus
		}

		time.Sleep(time.Duration(2) * time.Minute)
		// TODO: instead of sleeping for random/specified duration, use tsdb metrics
	}
}

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prombench-Restarter tool")
	app.HelpFlag.Short('h')

	s := newRestart()

	k8sApp := app.Command("restart", "Restart a Kubernetes deployment object \nex: ./restarter restart").
		Action(s.restart)
	k8sApp.Flag("vars", "When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.").
		Short('v').
		StringMapVar(&s.k8sClient.DeploymentVars)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}
}
