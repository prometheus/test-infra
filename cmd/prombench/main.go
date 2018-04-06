package main

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

	g := &gke.Cluster{}
	k8sGKE := app.Command("gke", "using the google container engine provider").Action(g.New)
	k8sGKE.Flag("project", "project name for the cluster").Default("prometheus").StringVar(&g.ProjectID)
	k8sGKE.Flag("zone", "zone for the cluster").Default("europe-west1-b").StringVar(&g.Zone)
	k8sGKE.Flag("name", "cluster name").Default("prombench").StringVar(&g.Name)
	k8sGKE.Flag("nodeCount", "total number of cluster nodes").Default("1").Int32Var(&g.NodeCount)
	k8sGKE.Flag("dashboard", "enable the dashboard").Default("false").BoolVar(&g.Dashboard)
	k8sGKE.Command("create", "create a new k8sGKE cluster").Action(g.Create)
	k8sGKE.Command("delete", "delete a k8sGKE cluster").Action(g.Delete)
	k8sGKE.Command("list", "list k8sGKE clusters").Action(g.List)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

}
