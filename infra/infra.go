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

package main // import "github.com/prometheus/test-infra/infra"

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/prometheus/test-infra/pkg/provider"
	"github.com/prometheus/test-infra/pkg/provider/eks"
	"github.com/prometheus/test-infra/pkg/provider/gke"
	kind "github.com/prometheus/test-infra/pkg/provider/kind"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)

	dr := provider.NewDeploymentResource()

	app := kingpin.New(filepath.Base(os.Args[0]), "The prometheus/test-infra deployment tool")
	app.HelpFlag.Short('h')
	app.Flag("file", "yaml file or folder  that describes the parameters for the object that will be deployed.").
		Short('f').
		ExistingFilesOrDirsVar(&dr.DeploymentFiles)
	app.Flag("vars", "When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.").
		Short('v').
		StringMapVar(&dr.FlagDeploymentVars)

	g := gke.New(dr)
	k8sGKE := app.Command("gke", `Google container engine provider - https://cloud.google.com/kubernetes-engine/`).
		Action(g.SetupDeploymentResources)
	k8sGKE.Flag("auth", "json authentication for the project. Accepts a filepath or an env variable that inlcudes tha json data. If not set the tool will use the GOOGLE_APPLICATION_CREDENTIALS env variable (export GOOGLE_APPLICATION_CREDENTIALS=service-account.json). https://cloud.google.com/iam/docs/creating-managing-service-account-keys.").
		PlaceHolder("service-account.json").
		Short('a').
		StringVar(&g.Auth)

	k8sGKE.Command("info", "gke info -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(g.GetDeploymentVars)

	// Cluster operations.
	k8sGKECluster := k8sGKE.Command("cluster", "manage GKE clusters").
		Action(g.NewGKEClient).
		Action(g.GKEDeploymentsParse)
	k8sGKECluster.Command("create", "gke cluster create -a service-account.json -f FileOrFolder").
		Action(g.ClusterCreate)
	k8sGKECluster.Command("delete", "gke cluster delete -a service-account.json -f FileOrFolder").
		Action(g.ClusterDelete)

	// Cluster node-pool operations
	k8sGKENodePool := k8sGKE.Command("nodes", "manage GKE clusters nodepools").
		Action(g.NewGKEClient).
		Action(g.GKEDeploymentsParse)
	k8sGKENodePool.Command("create", "gke nodes create -a service-account.json -f FileOrFolder").
		Action(g.NodePoolCreate)
	k8sGKENodePool.Command("delete", "gke nodes delete -a service-account.json -f FileOrFolder").
		Action(g.NodePoolDelete)
	k8sGKENodePool.Command("check-running", "gke nodes check-running -a service-account.json -f FileOrFolder").
		Action(g.AllNodepoolsRunning)
	k8sGKENodePool.Command("check-deleted", "gke nodes check-deleted -a service-account.json -f FileOrFolder").
		Action(g.AllNodepoolsDeleted)

	// K8s resource operations.
	k8sGKEResource := k8sGKE.Command("resource", `Apply and delete different k8s resources - deployments, services, config maps etc.Required variables -v GKE_PROJECT_ID, -v ZONE: -west1-b -v CLUSTER_NAME`).
		Action(g.NewGKEClient).
		Action(g.K8SDeploymentsParse).
		Action(g.NewK8sProvider)
	k8sGKEResource.Command("apply", "gke resource apply -a service-account.json -f manifestsFileOrFolder -v GKE_PROJECT_ID:test -v ZONE:europe-west1-b -v CLUSTER_NAME:test -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(g.ResourceApply)
	k8sGKEResource.Command("delete", "gke resource delete -a service-account.json -f manifestsFileOrFolder -v GKE_PROJECT_ID:test -v ZONE:europe-west1-b -v CLUSTER_NAME:test -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(g.ResourceDelete)

	k := kind.New(dr)
	k8sKIND := app.Command("kind", `Kubernetes In Docker (KIND) provider - https://kind.sigs.k8s.io/docs/user/quick-start/`).
		Action(k.SetupDeploymentResources)

	k8sKIND.Command("info", "kind info -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(k.GetDeploymentVars)

	//Cluster operations.
	k8sKINDCluster := k8sKIND.Command("cluster", "manage KIND clusters").
		Action(k.KINDDeploymentsParse)
	k8sKINDCluster.Command("create", "kind cluster create -f File -v PR_NUMBER:$PR_NUMBER -v CLUSTER_NAME:$CLUSTER_NAME").
		Action(k.ClusterCreate)
	k8sKINDCluster.Command("delete", "kind cluster delete -f File -v PR_NUMBER:$PR_NUMBER -v CLUSTER_NAME:$CLUSTER_NAME").
		Action(k.ClusterDelete)

	// K8s resource operations.
	k8sKINDResource := k8sKIND.Command("resource", `Apply and delete different k8s resources - deployments, services, config maps etc.`).
		Action(k.NewK8sProvider).
		Action(k.K8SDeploymentsParse)
	k8sKINDResource.Command("apply", "kind resource apply -f manifestsFileOrFolder -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(k.ResourceApply)
	k8sKINDResource.Command("delete", "kind resource delete -f manifestsFileOrFolder -v hashStable:COMMIT1 -v hashTesting:COMMIT2").
		Action(k.ResourceDelete)

	// EKS based commands
	e := eks.New()
	k8sEKS := app.Command("eks", "Amazon Elastic Kubernetes Service - https://aws.amazon.com/eks").
		Action(e.NewEKSClient)
	k8sEKS.Flag("file", "yaml file or folder  that describes the parameters for the object that will be deployed.").
		Required().
		Short('f').
		ExistingFilesOrDirsVar(&e.DeploymentFiles)
	k8sEKS.Flag("vars", "When provided it will substitute the token holders in the yaml file. Follows the standard golang template formating - {{ .hashStable }}.").
		Short('v').
		StringMapVar(&e.DeploymentVars)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

}
