package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/providers/gke"
	"gopkg.in/alecthomas/kingpin.v2"
)

type k8sGKECluster interface {
	Create(*kingpin.ParseContext) error
	// Delete()
	// Scale()
}

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prometheus benchmarking tool")
	app.HelpFlag.Short('h')

	g := &gke.Cluster{}
	k8sGKE := app.Command("gke", "using the google container engine provider").Action(g.New)
	k8sGKE.Flag("project", "the project name for the cluster").Default("prometheus").StringVar(&g.ProjectID)
	k8sGKE.Flag("zone", "the zone for the cluster").Default("europe-west1-b").StringVar(&g.Zone)
	k8sGKE.Flag("name", "the cluster name").Default("prombench").StringVar(&g.Name)
	k8sGKE.Flag("nodeCount", "the total number of cluster nodes").Default("1").Int64Var(&g.NodeCount)
	k8sGKE.Command("create", "create a new k8sGKE cluster").Action(g.Create)
	k8sGKE.Command("delete", "delete a k8sGKE cluster").Action(g.Delete)
	k8sGKE.Command("list", "list k8sGKE clusters").Action(g.List)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

}
