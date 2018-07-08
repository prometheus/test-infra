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
	k8sGKE := app.Command("gke", `Google container engine provider - https://cloud.google.com/kubernetes-engine/`).
		Action(g.NewGKEClient).
		Action(g.ConfigParse)
	k8sGKE.Flag("auth", "json authentication file for the project - https://cloud.google.com/iam/docs/creating-managing-service-account-keys. If not set the tool will use the GOOGLE_APPLICATION_CREDENTIALS env variable (export GOOGLE_APPLICATION_CREDENTIALS=service-account.json)").
		Required().
		PlaceHolder("service-account.json").
		Short('a').
		ExistingFileVar(&g.AuthFile)
	k8sGKE.Flag("config", "GKE cluster-config yaml file").
		PlaceHolder("cluster.yaml").
		Short('c').
		Default("config/cluster.yaml").
		ExistingFileVar(&g.ConfigFile)
	k8sGKE.Flag("file", "yaml file used to apply or delete k8s resources. If directory is given, all the yaml files from are read recursively from it.").
		PlaceHolder("resources.yaml").
		Short('f').
		Default("manifests").
		ExistingFilesOrDirsVar(&g.ResourceFiles)
	k8sGKE.Flag("vars", "When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.").
		Short('v').
		StringMapVar(&g.ResourceVars)

	// cluster operations
	k8sGKECluster := k8sGKE.Command("cluster", "Create or delete GKE k8s clusters")
	k8sGKECluster.Command("create", "gke cluster create -a service-account.json -c config/cluster.yaml").
		Action(g.ClusterCreate)
	k8sGKECluster.Command("delete", "gke cluster delete -a service-account.json -c config/cluster.yaml").
		Action(g.ClusterDelete)

	// node-pool operations
	k8sGKENodePool := k8sGKE.Command("nodepool", "Scale up or down a k8s clusters using node-pools")
	k8sGKENodePool.Command("create", "gke nodepool create -a service-account.json -c config/cluster.yaml").
		Action(g.NodePoolCreate)
	k8sGKENodePool.Command("delete", "gke nodepool delete -a service-account.json -c config/cluster.yaml").
		Action(g.NodePoolDelete)

	k8sGKEResource := k8sGKE.Command("resource", "Create,update and delete different k8s resources - deployments, services, config maps etc.").
		Action(g.NewResourceClient)
	k8sGKEResource.Command("apply", "gke resource apply -a service-account.json -c config/cluster.yaml -f manifests -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(g.ResourceApply)
	k8sGKEResource.Command("delete", "gke resource delete -a service-account.json -c config/cluster.yaml -f manifests -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(g.ResourceDelete)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

}
