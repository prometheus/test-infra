// Copyright 2019 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main // import "github.com/prometheus/prombench/cmd/prombench"

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/pkg/provider/gke"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prometheus benchmarking tool")
	app.HelpFlag.Short('h')

	g := gke.New()
	k8sGKE := app.Command("gke", `Google container engine provider - https://cloud.google.com/kubernetes-engine/`).
		Action(g.NewGKEClient)
	k8sGKE.Flag("auth", "json authentication for the project. Accepts a filepath or an env variable that inlcudes tha json data. If not set the tool will use the GOOGLE_APPLICATION_CREDENTIALS env variable (export GOOGLE_APPLICATION_CREDENTIALS=service-account.json). https://cloud.google.com/iam/docs/creating-managing-service-account-keys.").
		PlaceHolder("service-account.json").
		Short('a').
		StringVar(&g.Auth)
	k8sGKE.Flag("file", "yaml file or folder  that describes the parameters for the object that will be deployed.").
		Required().
		Short('f').
		ExistingFilesOrDirsVar(&g.DeploymentFiles)
	k8sGKE.Flag("vars", "When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.").
		Short('v').
		StringMapVar(&g.DeploymentVars)

	// K8s ConfigMap operations from file.
	k8sConfigMap := k8sGKE.Command("configmap", "use configmap").
		Action(g.NewK8sProvider).
		Action(g.K8SDeploymentsParse)
	k8sConfigMap.Flag("enabled", "enable creation of ConfigMap from file").
		Hidden().
		Default("true").
		BoolVar(&g.ConfigMapConfig.Enabled)
	k8sConfigMap.Flag("name", "Name of the ConfigMap").Required().StringVar(&g.ConfigMapConfig.Name)
	k8sConfigMap.Flag("namespace", "Namespace of the ConfigMap").Required().StringVar(&g.ConfigMapConfig.Namespace)

	k8sConfigMap.Command("apply", "apply the configmap").
		Action(g.ResourceApply)
	k8sConfigMap.Command("delete", "delete the configmap").
		Action(g.ResourceDelete)

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
	k8sGKENodePool.Command("check-running", "gke nodepool check-running -a service-account.json -f FileOrFolder").
		Action(g.AllNodepoolsRunning)
	k8sGKENodePool.Command("check-deleted", "gke nodepool check-deleted -a service-account.json -f FileOrFolder").
		Action(g.AllNodepoolsDeleted)

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
