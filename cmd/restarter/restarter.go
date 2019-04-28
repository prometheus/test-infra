package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	appsV1 "k8s.io/api/apps/v1"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/pkg/provider/k8s"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/apimachinery/pkg/runtime"
)

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

func (s *restart) updateReplicas(restartCounter int) []k8s.Resource {
	var k8sResource []k8s.Resource
	for _, deployment := range s.k8sClient.GetResourses() {
		k8sObjects := make([]runtime.Object, 0)

		for _, resource := range deployment.Objects {
			if kind := strings.ToLower(resource.GetObjectKind().GroupVersionKind().Kind); kind == "deployment" {
				req := resource.(*appsV1.Deployment)
				req.Spec.Template.ObjectMeta.Labels["restart_count"] = fmt.Sprintf("%v", restartCounter)
				k8sObjects = append(k8sObjects, req.DeepCopyObject())
			}
		}
		if len(k8sObjects) > 0 {
			k8sResource = append(k8sResource, k8s.Resource{FileName: deployment.FileName, Objects: k8sObjects})
		}
	}
	return k8sResource
}

func (s *restart) restart(*kingpin.ParseContext) error {
	log.Printf("Starting Prombench-Restarter")

	restartCounter := 0

	// TODO:
	// a polling mechanism to see if there were any compaction failure in the
	// last minute then trigger restart if true then sleep for a minute

	for {
		restartCounter++
		updatedResources := s.updateReplicas(restartCounter)
		if err := s.k8sClient.ResourceApply(updatedResources); err != nil {
			fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error updating deployment"))
		}

		time.Sleep(time.Duration(3) * time.Minute)
	}
}

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prombench-Restarter tool")
	app.HelpFlag.Short('h')

	s := newRestart()

	k8sApp := app.Command("restart", "Restart a Kubernetes deployment object \nex: ./restarter restart").
		Action(s.k8sClient.DeploymentsParse).
		Action(s.restart)
	k8sApp.Flag("file", "yaml file or folder that describes the parameters for the deployment.").
		Required().
		Short('f').
		ExistingFilesOrDirsVar(&s.k8sClient.DeploymentFiles)
	k8sApp.Flag("vars", "When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.").
		Short('v').
		StringMapVar(&s.k8sClient.DeploymentVars)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}
}
