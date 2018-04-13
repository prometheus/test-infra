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

	g := gke.New()
	k8sGKE := app.Command("gke", "using the google container engine provider").Action(g.NewGKEClient).Action(g.ConfigParse)
	k8sGKE.Flag("config", "yaml GKE config file for the cluster and nodes").Short('c').Required().ExistingFileVar(&g.ClusterConfigFile)
	k8sGKE.Flag("auth", "json authentication file for the project - https://cloud.google.com/iam/docs/creating-managing-service-account-keys. If not set it will use the GOOGLE_APPLICATION_CREDENTIALS env variable").Short('a').ExistingFileVar(&g.AuthFile)

	k8sGKECluster := k8sGKE.Command("cluster", "cluster commands")
	k8sGKECluster.Command("create", "create a new k8s cluster").Action(g.ClusterCreate)
	k8sGKECluster.Command("delete", "delete a k8s cluster").Action(g.ClusterDelete)

	k8sGKEResource := k8sGKE.Command("resource", "create or update different k8s resources").Action(g.NewResourceClient)
	k8sGKEResource.Flag("file", "resources file").Short('f').Required().ExistingFilesVar(&g.ResourceFiles)
	k8sGKEResource.Flag("vars", "variables to substitute in the resources file").Short('v').Required().StringMapVar(&g.ResourceVars)
	k8sGKEResource.Command("delete", "delete a k8s resources").Action(g.ResourceDelete)
	k8sGKEResource.Command("apply", "create or update a k8s resources").Action(g.ResourceApply)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

}
