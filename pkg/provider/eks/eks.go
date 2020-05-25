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

package eks

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	eks "github.com/aws/aws-sdk-go/service/eks"
	"github.com/pkg/errors"
	k8sProvider "github.com/prometheus/test-infra/pkg/provider/k8s"

	"github.com/prometheus/test-infra/pkg/provider"
	"gopkg.in/alecthomas/kingpin.v2"
	yamlGo "gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

type Resource = provider.Resource

type eksCluster struct {
	Cluster    eks.CreateClusterInput
	NodeGroups []eks.CreateNodegroupInput
}

// EKS holds the fields used to generate an API request.
type EKS struct {
	ClusterName string
	// The eks client used when performing EKS requests.
	clientEKS *eks.EKS
	// The k8s provider used when we work with the manifest files.
	k8sProvider *k8sProvider.K8s
	// DeploymentFiles files provided from the cli.
	DeploymentFiles []string
	// Variables to substitute in the DeploymentFiles.
	// These are also used when the command requires some variables that are not provided by the deployment file.
	DeploymentVars map[string]string
	// These works in the same way as DelpoymentVars but they are used to pass on stringified range value to deployment file.
	DeploymentRangeVars map[string]string
	// These string is used to split DeploymentRangeVars in to their corresponding lists.
	Separator string
	// Content bytes after parsing the template variables, grouped by filename.
	eksResources []Resource
	// K8s resource.runtime objects after parsing the template variables, grouped by filename.
	k8sResources []k8sProvider.Resource
}

// New is the EKS constructor
func New() *EKS {
	return &EKS{
		DeploymentVars:      make(map[string]string),
		DeploymentRangeVars: make(map[string]string),
		Separator:           "_",
	}
}

// NewEKSClient sets the EKS client used when performing the GKE requests.
func (c *EKS) NewEKSClient(*kingpin.ParseContext) error {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return errors.Errorf("no auth provided! Need to set the AWS_ACCCESS_KEY_ID and AWS_SECRET_ACCESS_KEY env variable")
	}

	cl := eks.New(awsSession.Must(awsSession.NewSession()), aws.NewConfig())
	c.clientEKS = cl
	return nil
}

func (c *EKS) eksDeploymentVars() map[string]string {
	deployVars := c.DeploymentVars

	for key, rangeVars := range c.DeploymentRangeVars {
		deployVars[key] = rangeVars
	}

	deployVars["SEPERATOR"] = c.Separator

	return deployVars
}

// EKSDeploymentParse parses the cluster/nodegroups deployment file and saves the result as bytes grouped by the filename.
// Any variables passed to the cli will be replaced in the resource files following the golang text template format.
func (c *EKS) EKSDeploymentParse(*kingpin.ParseContext) error {

	deploymentResource, err := provider.DeploymentsParse(c.DeploymentFiles, c.eksDeploymentVars())
	if err != nil {
		log.Fatalf("Couldn't parse deployment files: %v", err)
	}

	c.eksResources = deploymentResource
	return nil
}

// K8SDeploymentsParse parses the k8s objects deployment files and saves the result as k8s objects grouped by the filename.
// Any variables passed to the cli will be replaced in the resources files following the golang text template format.
func (c *EKS) K8SDeploymentsParse(*kingpin.ParseContext) error {
	deploymentResource, err := provider.DeploymentsParse(c.DeploymentFiles, c.DeploymentVars)
	if err != nil {
		log.Fatalf("Couldn't parse deployment files: %v", err)
	}

	for _, deployment := range deploymentResource {

		decode := scheme.Codecs.UniversalDeserializer().Decode
		k8sObjects := make([]runtime.Object, 0)

		for _, text := range strings.Split(string(deployment.Content), provider.Separator) {
			text = strings.TrimSpace(text)
			if len(text) == 0 {
				continue
			}

			resource, _, err := decode([]byte(text), nil, nil)
			if err != nil {
				return errors.Wrapf(err, "decoding the resource file:%v, section:%v...", deployment.FileName, text[:100])
			}
			if resource == nil {
				continue
			}
			k8sObjects = append(k8sObjects, resource)
		}
		if len(k8sObjects) > 0 {
			c.k8sResources = append(c.k8sResources, k8sProvider.Resource{FileName: deployment.FileName, Objects: k8sObjects})
		}
	}
	return nil
}

// ClusterCreate create a new cluster or applies changes to an existing cluster.
func (c *EKS) ClusterCreate(*kingpin.ParseContext) error {
	req := eks.CreateClusterInput{}
	for _, deployment := range c.eksResources {

		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		log.Printf("Cluster create request: name:'%v'", req.Name)
		_, err := c.clientEKS.CreateCluster(&req)
		if err != nil {
			log.Fatalf("Couldn't create cluster '%v', file:%v ,err: %v", req.Name, deployment.FileName, err)
		}

		// err = provider.RetryUntilTrue(
		// 	fmt.Sprintf("creating cluster:%v", req.Name),
		// 	provider.GlobalRetryCount,
		// 	func() (bool, error) { return c.clusterRunning(req.Zone, req.ProjectId, req.Cluster.Name) })

		if err != nil {
			log.Fatalf("creating cluster err:%v", err)
		}
	}
	return nil
}

// ClusterDelete deletes a eks Cluster
func (c *EKS) ClusterDelete(*kingpin.ParseContext) error {
	req := eks.CreateClusterInput{}
	for _, deployment := range c.eksResources {

		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		reqD := &eks.DeleteClusterInput{
			Name: req.Name,
		}
		log.Printf("Removing cluster '%v'", reqD.Name)

		// err := provider.RetryUntilTrue(
		// 	fmt.Sprintf("deleting cluster:%v", reqD.ClusterId),
		// 	provider.GlobalRetryCount,
		// 	func() (bool, error) { return c.clusterDeleted(reqD) })

		// if err != nil {
		// 	log.Fatalf("removing cluster err:%v", err)
		// }
	}
	return nil
}

// clusterRunning checks whether a cluster is in a active state.
func (c *EKS) clusterRunning(name string) (bool, error) {
	req := &eks.DescribeClusterInput{
		Name: aws.String(name),
	}
	clusterRes, err := c.clientEKS.DescribeCluster(req)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == eks.ErrCodeNotFoundException {
			return false, nil
		}
		return false, fmt.Errorf("Couldn't get cluster status: %v", err)
	}
	if *clusterRes.Cluster.Status == eks.ClusterStatusFailed {
		return false, fmt.Errorf("Cluster not in a status to become ready - %s", *clusterRes.Cluster.Status)
	}
	if *clusterRes.Cluster.Status == eks.ClusterStatusActive {
		return true, nil
	}
	log.Printf("Cluster '%v' status: %v", name, *clusterRes.Cluster.Status)
	return false, nil
}
