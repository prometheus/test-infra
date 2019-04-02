package main

import (
	"context"
	"fmt"
	"log"
	//"math/rand"
	"os"
	"path/filepath"
	//"strings"
	"time"

	//appsV1 "k8s.io/api/apps/v1"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/pkg/provider/k8s"
	"gopkg.in/alecthomas/kingpin.v2"
	//"k8s.io/apimachinery/pkg/runtime"
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

func (s *restart) restart(*kingpin.ParseContext) error {
	log.Printf("Starting Prombench-Restarter")

	prNo := s.k8sClient.DeploymentVars["PR_NUMBER"]
	namespace := "prombench-" + prNo
	pods, err := s.k8sClient.FetchCurrentPods(namespace,"app=prometheus")
	killcommand := "/bin/kill -9 $(pidof prometheus)"
	prDir := "/go/src/github.com/prometheus/prometheus"

	runforPr := fmt.Sprintf(`cd %s ;
	./prometheus --config.file=/etc/prometheus/prometheus.yml \
             --storage.tsdb.path=/prometheus \
             --web.console.libraries=%s/console_libraries \
             --web.console.templates=%s/consoles \
             --web.external-url=http://prombench.prometheus.io/%s/prometheus-pr \
             --log.level=debug`, prDir, prDir, prDir, prNo)
	runforRelease := fmt.Sprintf(`/bin/prometheus --config.file=/etc/prometheus/prometheus.yml \
          --storage.tsdb.path=/prometheus \
          --web.console.libraries=/etc/prometheus/console_libraries \
          --web.console.templates=/etc/prometheus/consoles \
          --web.external-url=http://prombench.prometheus.io/%s/prometheus-release \
          --log.level=debug`, prNo)

	if err != nil {
		log.Printf("Error fetching pods: %v", err)
	}

	for {
		for _, pod := range pods.Items {
			_, err := s.k8sClient.ExecuteInPod(killcommand, pod.ObjectMeta.Name, "prometheus", namespace)
			if err != nil {
				log.Printf("Error executing command: %v", err)
			}
		}
		// wait for sometime maybe
		log.Printf("sleepin 1 min")
		time.Sleep(time.Duration(1) * time.Minute)
		for _, pod := range pods.Items {

			if pod.ObjectMeta.Labels["prometheus"][:7] == "test-pr" {
				_, err := s.k8sClient.ExecuteInPod(runforPr, pod.ObjectMeta.Name, "prometheus", namespace)
				if err != nil { log.Printf("Error executing command: %v", err) }
			} else {
				_, err := s.k8sClient.ExecuteInPod(runforRelease, pod.ObjectMeta.Name, "prometheus", namespace)
				if err != nil { log.Printf("Error executing command: %v", err) }
			}

		}

		log.Printf("sleepin 1 min again")
		time.Sleep(time.Duration(1) * time.Minute)
		//time.Sleep(time.Duration(rand.Intn(20) + 10) * time.Minute)
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
