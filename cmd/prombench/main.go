package main // import "github.com/prometheus/prombench/cmd/prombench"

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/provider/gke"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prometheus benchmarking tool")
	app.HelpFlag.Short('h')

	g := &gke.GKE{}
	k8sGKE := app.Command("gke", "using the google container engine provider").Action(g.NewGKEClient)
	k8sGKE.Flag("config", "the GKE config file for the cluster and nodes").Short('c').Required().ExistingFileVar(&g.ConfigFile)

	k8sGKECluster := k8sGKE.Command("cluster", "cluster commands")
	k8sGKECluster.Command("create", "create a new k8s cluster").Action(g.ClusterCreate)
	k8sGKECluster.Command("delete", "delete a k8s cluster").Action(g.ClusterDelete)
	k8sGKECluster.Command("list", "list k8s clusters").Action(g.ClusterList)
	k8sGKECluster.Command("get", "get details for a k8s cluster").Action(g.ClusterGet)

	// k8sGKEDeployment := k8sGKE.Command("deployment", "deployment commands").Action(g.NewDeploymentClient)
	// k8sGKEDeployment.Flag("file", "deployment manifest file").Short('f').Required().ExistingFilesVar(&g.Deployments)
	// k8sGKEDeployment.Flag("vars", "deployment manifest file").Short('v').Required().StringMapVar(&g.Deployments)
	// k8sGKEDeployment.Command("apply", "apply a k8s deployment manifest").Action(g.DeploymentApply)
	// k8sGKEDeployment.Command("delete", "delete a k8s deployment").Action(g.DeploymentDelete)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

}
