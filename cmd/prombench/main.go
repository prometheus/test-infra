package main // import "github.com/prometheus/prombench/cmd/prombench"

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

	g := gke.New()
	k8sGKE := app.Command("gke", `Google container engine provider - https://cloud.google.com/kubernetes-engine/.Requires`).
		Action(g.NewGKEClient).
		Action(g.ConfigParse)
	k8sGKE.Flag("config", "yaml GKE config file used to create or delete the k8s cluster and nodes").
		Default("../../config/cluster.yaml").
		PlaceHolder("cluster.yaml").
		Short('c').
		ExistingFileVar(&g.ClusterConfigFile)
	k8sGKE.Flag("auth", "json authentication file for the project - https://cloud.google.com/iam/docs/creating-managing-service-account-keys. If not set the tool will use the GOOGLE_APPLICATION_CREDENTIALS env variable (export GOOGLE_APPLICATION_CREDENTIALS=key.json)").
		PlaceHolder("key.json").
		Short('a').
		ExistingFileVar(&g.AuthFile)

	k8sGKECluster := k8sGKE.Command("cluster", "Create or delete k8s clusters")
	k8sGKECluster.Command("create", "gke cluster create -a key.json  -c ../../config/cluster.yaml").
		Action(g.ClusterCreate)
	k8sGKECluster.Command("delete", "gke cluster delete -a key.json  -c ../../config/cluster.yaml").
		Action(g.ClusterDelete)

	k8sGKEResource := k8sGKE.Command("resource", "Create,update and delete different k8s resources - deployments, services, config maps etc.").
		Action(g.NewResourceClient)
	k8sGKEResource.Flag("file", "yaml file used to apply or delete k8s resources. It uses the standard k8s formatting. It also supports the default golang templates.").
		Default("../../config/resources.yaml").
		PlaceHolder("resources.yaml").
		Short('f').
		ExistingFilesVar(&g.ResourceFiles)
	k8sGKEResource.Flag("vars", "When provided it will substitute the token holders in the resources file. Follows the standard golang template formating - {{ hashStable }}.").
		Short('v').
		StringMapVar(&g.ResourceVars)
	k8sGKEResource.Command("apply", "gke resource apply -a ../../config/key.json -c ../../config/cluster.yaml -f ../../config/resources.yaml --vars hashStable:COMMIT1 --vars hashTesting:COMMIT2").
		Action(g.ResourceApply)
	k8sGKEResource.Command("delete", "gke resource delete -a ../../config/key.json -c ../../config/cluster.yaml -f ../../config/resources.yaml --vars hashStable:COMMIT1 --vars hashTesting:COMMIT2").
		Action(g.ResourceDelete)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

}
