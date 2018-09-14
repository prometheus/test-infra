package main // import "github.com/prometheus/prombench/cmd/prombench"

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/pkg/provider/gke"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prometheus benchmarking tool")
	app.HelpFlag.Short('h')

	g := gke.New()
	k8sGKE := app.Command("gke", `Google container engine provider - https://cloud.google.com/kubernetes-engine/`).
		Action(g.NewGKEClient)
	k8sGKE.Flag("auth", "json authentication file for the project - https://cloud.google.com/iam/docs/creating-managing-service-account-keys. If not set the tool will use the GOOGLE_APPLICATION_CREDENTIALS env variable (export GOOGLE_APPLICATION_CREDENTIALS=service-account.json)").
		PlaceHolder("service-account.json").
		Short('a').
		ExistingFileVar(&g.AuthFile)
	k8sGKE.Flag("file", "yaml file or folder  that describes the parameters for the object that will be deployed.").
		Required().
		Short('f').
		ExistingFilesOrDirsVar(&g.DeploymentFiles)
	k8sGKE.Flag("vars", "When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.").
		Short('v').
		StringMapVar(&g.DeploymentVars)

	// Cluster operations.
	k8sGKECluster := k8sGKE.Command("cluster", "manage GKE clusters").
		Action(g.GKEDeploymentsParse)
	k8sGKECluster.Command("create", "gke cluster create -a service-account.json -f FileOrFolder").
		Action(g.ClusterCreate)
	k8sGKECluster.Command("delete", "gke cluster delete -a service-account.json -f FileOrFolder").
		Action(g.ClusterDelete)

	// Cluster node-pool operations
	k8sGKENodePool := k8sGKE.Command("nodepool", "manage GKE clusters nodepools").
		Action(g.GKEDeploymentsParse)
	k8sGKENodePool.Command("create", "gke nodepool create -a service-account.json -f FileOrFolder").
		Action(g.NodePoolCreate)
	k8sGKENodePool.Command("delete", "gke nodepool delete -a service-account.json -f FileOrFolder").
		Action(g.NodePoolDelete)

	// K8s resource operations.
	k8sGKEResource := k8sGKE.Command("resource", `Apply and delete different k8s resources - deployments, services, config maps etc.Required variables -v PROJECT_ID, -v ZONE: -west1-b -v CLUSTER_NAME`).
		Action(g.NewK8sProvider).
		Action(g.K8SDeploymentsParse)
	k8sGKEResource.Command("apply", "gke resource apply -a service-account.json -f manifestsFileOrFolder -v PROJECT_ID:test -v ZONE:europe-west1-b -v CLUSTER_NAME:test -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(g.ResourceApply)
	k8sGKEResource.Command("delete", "gke resource delete -a service-account.json -f manifestsFileOrFolder -v PROJECT_ID:test -v ZONE:europe-west1-b -v CLUSTER_NAME:test -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(g.ResourceDelete)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

}
