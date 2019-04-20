package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/pkg/provider/k8s"
	"gopkg.in/alecthomas/kingpin.v2"
	apiCoreV1 "k8s.io/api/core/v1"
)

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

func killPrometheus(c chan string, wg sync.WaitGroup, pod *apiCoreV1.Pod, namespace string) {
	defer wg.Done()
	command := "/scripts/killer.sh"

	// maybe we should be checking pod.status.Conditions tYpe instead
	if pod.Status.Phase != "Running" {
		// cancel all goroutines in c
		c = nil
		log.Fatalf("All pods not ready")
	}

	c <- pod.ObjectMeta.Name

	resp, err := s.k8sClient.ExecuteInPod(command, pod.ObjectMeta.Name, "prometheus", namespace)
	if err != nil {
		// failed
		// Q: should probably add a metric here?
	}
	// check response status code
}

func (s *restart) restart(*kingpin.ParseContext) error {
	log.Printf("Starting Prombench-Restarter")

	prNo := s.k8sClient.DeploymentVars["PR_NUMBER"]
	namespace := "prombench-" + prNo
	podList, err := s.k8sClient.FetchRunningPods(namespace, "app=prometheus")

	// if not exit so that the restarter is restarted

	if err != nil {
		log.Fatalf("Error fetching pods: %v", err)
	}

	if len(pods) != 2 {
		log.Fatalf("All pods not ready")
	}

	podsToKill := make(chan string)
	//podsToKill := make(chan string, 2)
	//podsToStart := make(chan string)

	for {
		var wgKill sync.WaitGroup
		var wgRestart sync.WaitGroup

		for _, pod := range podList.Items {
			wg.Add(1)
			go killPrometheus(podsToKill, wgKill, pod, namespace)
		}
		close(podsToKill) // do we kill a buffered channel, does that start togerter thing work
		// with buffered channels
		wgKill.Wait()
		// trying to print podsToKill should panic unless I make it a buffered channel

		// we need something to start the killing at the same time
		// after that we need something that will tell us when the killing is done

		//for _, pod := range pods.Items {
		//	go killPrometheus(killingDone, pod, namespace)
		//}

		// Sleep for amount of time 10 >= n <= 30 mins, then restart both
		time.Sleep(time.Duration(rand.Intn(20)+10) * time.Minute)
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
