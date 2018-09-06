package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/pkg/provider/k8s"
	"gopkg.in/alecthomas/kingpin.v2"
)

type scale struct {
	k8sClient         *k8s.K8sClient
	scaleUpReplicas   int32
	scaleDownReplicas int32
	intervalMinutes   int
}

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prombench-Scaler tool")
	app.HelpFlag.Short('h')

	k, err := k8s.NewK8sClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error creating k8s client inside the k8s cluster"))
		os.Exit(2)
	}

	s := scale{k8sClient: k}

	k8sApp := app.Command("scale", `Scale a Kubernetes deployment object periodically up and down`).
		Action(s.k8sClient.DeploymentsParse)
	k8sApp.Flag("file", "yaml file or folder that describes the parameters for the deployment.").
		Required().
		Short('f').
		ExistingFilesOrDirsVar(&s.k8sClient.DeploymentFiles)
	k8sApp.Flag("vars", "When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.").
		Short('v').
		StringMapVar(&s.k8sClient.DeploymentVars)
	k8sApp.Flag("scaleUpReplicas", "Number of Replicas to scale up the deployment.").
		Short('u').
		Int32Var(&s.scaleUpReplicas)
	k8sApp.Flag("scaleDownReplicas", "Number of Replicas to scale down the deployment.").
		Short('d').
		Int32Var(&s.scaleDownReplicas)
	k8sApp.Flag("intervalMinutes", "Time to wait(in minutes) before changing the number of replicas.").
		Short('i').
		IntVar(&s.intervalMinutes)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

	log.Printf("Starting Prombench-Scaler")
	for {
		time.Sleep(time.Duration(s.intervalMinutes) * time.Minute)

		log.Printf("Scaling Deployment to %d", s.scaleDownReplicas)
		if err := s.k8sClient.Scale(&s.scaleDownReplicas); err != nil {
			fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error scaling deployment"))
			os.Exit(2)
		}

		time.Sleep(time.Duration(s.intervalMinutes) * time.Minute)

		log.Printf("Scaling Deployment to %d", s.scaleUpReplicas)
		if err := s.k8sClient.Scale(&s.scaleUpReplicas); err != nil {
			fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error scaling deployment"))
			os.Exit(2)
		}
	}
}
