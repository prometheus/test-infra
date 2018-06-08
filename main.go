package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/benchmark/provider/gke"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prometheus benchmarking tool")
	app.HelpFlag.Short('h')

	g := &gke.GKE{}

	k8sGKE := app.Command("benchmark", "Using the Google Kubernetes Engine provider").Action(g.start)
	k8sGKE.Flag("project", "project name for the cluster").Required().StringVar(&g.ProjectID)
	k8sGKE.Flag("zone", "zone for the cluster").Required().StringVar(&g.Zone)
	k8sGKE.Flag("cluster-name", "name of cluster").Required().StringVar(&g.Name)
	k8sGKE.Flag("cluster-name", "name of cluster").Required().StringVar(&g.Name)
	k8sGKE.Flag("cluster-name", "name of cluster").Required().StringVar(&g.Name)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}
}
