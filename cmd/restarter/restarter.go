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

// TODO: can we make the filds internal?
type operator struct {
	Context context.Context
	Cancel  context.CancelFunc
	Blocker chan struct{}
	Err     error // make it [] to hold errors for all the operations
	Wg      sync.WaitGroup
}

type restart struct {
	k8sClient *k8s.K8s
}

func new() *restart {
	k, err := k8s.New(context.Background(), nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error creating k8s client inside the k8s cluster"))
		os.Exit(2)
	}
	return &restart{
		k8sClient: k,
	}
}

func (s *restart) killPrometheus(o *operator, ready chan struct{}, pod apiCoreV1.Pod, namespace string) {
	log.Println("killPrometheus Running")
	defer o.Wg.Done()
	command := "/scripts/killer.sh"
	container := "prometheus"
	cStatus := podV1.GetExistingContainerStatus(pod.Status.ContainerStatuses, container)

	if !cStatus.Ready {
		o.Err = fmt.Errorf("containers not ready for pod: %v", pod.ObjectMeta.Name)
	}

	ready <- struct{}{} // block

	select {
	case <-o.Context.Done():
		log.Printf("cancelling now %v\n", pod.ObjectMeta.Name)
		return
	case <-o.Blocker:
		resp, err := s.k8sClient.ExecuteInPod(command, pod.ObjectMeta.Name, container, namespace)
		if err != nil {
			o.Err = fmt.Errorf("Kill command failed with: ", pod.ObjectMeta.Name)
			// we can use pkg here
		}
		return
	}

}

func (s *restart) startPrometheus(o *operator, ready chan struct{}, pod apiCoreV1.Pod, namespace string) {
	log.Println("startPrometheus Running")
	defer o.Wg.Done()
	command := "/scripts/restarter.sh"
	container := "prometheus"
	cStatus := podV1.GetExistingContainerStatus(pod.Status.ContainerStatuses, container)

	if cStatus.Ready {
		o.Err = fmt.Errorf("containers is ready, do not restart if already ready: %v", pod.ObjectMeta.Name)
	}

	ready <- struct{}{} // block
	log.Printf("blocking now %v\n", pod.ObjectMeta.Name)

	select {
	case <-o.Context.Done():
		log.Printf("cancelling now %v\n", pod.ObjectMeta.Name)
		return
	case <-o.Blocker:
		resp, err := s.k8sClient.ExecuteInPod(command, pod.ObjectMeta.Name, container, namespace)
		if err != nil {
			o.Err = fmt.Errorf("Start command failed with: %v", pod.ObjectMeta.Name)
			// we can use pkg here
		}
		return
	}
}

func (s *restart) restart(*kingpin.ParseContext) error {
	log.Printf("Starting Prombench-Restarter")

	prNo := s.k8sClient.DeploymentVars["PR_NUMBER"]
	namespace := "prombench-" + prNo
	ready := make(chan struct{})
	restartCount := 0

	for {
		if restartCount != 0 {
			time.Sleep(time.Duration(300) * time.Second)
		}
		restartCount += 1 // maybe add a metric here

		var killer, starter operator
		killer.Blocker = make(chan struct{})
		killer.Context, killer.Cancel = context.WithCancel(context.Background())
		starter.Blocker = make(chan struct{})
		starter.Context, starter.Cancel = context.WithCancel(context.Background())

		// get podList everytime because podName can change
		podList, err := s.k8sClient.FetchRunningPods(namespace, "app=prometheus", "")
		if err != nil {
			log.Fatalf("Error fetching pods: %v", err)
		}
		if len(podList.Items) != 2 {
			log.Fatalf("All pods not ready")
		}

		// kill prometheus
		for _, pod := range podList.Items {
			killer.Wg.Add(1)
			go s.killPrometheus(&killer, ready, pod, namespace)
			<-ready
		}
		if killer.Err != nil {
			killer.Cancel()
			log.Println(killer.Err)
			continue
			// instead of continuing we can try starting
		}
		close(killer.Blocker)
		killer.Wg.Wait()
		if killer.Err != nil {
			// wss error, http response err, unix return code err
			log.Println(killer.Err)
			// this means that one or both(after []error) had issues with killing
		}

		// start prometheus
		for _, pod := range podList.Items {
			starter.Wg.Add(1)
			go s.startPrometheus(&starter, ready, pod, namespace)
			<-ready
		}
		if starter.Err != nil {
			starter.Cancel()
			log.Println(starter.Err)
			continue
		}
		close(starter.Blocker)
		starter.Wg.Wait()
		if starter.Err != nil {
			log.Println(starter.Err)
		}
	}
}

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prombench-Restarter tool")
	app.HelpFlag.Short('h')

	s := new()

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
