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
	"github.com/aws/aws-sdk-go/aws/credentials"
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
	AuthFilename string

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
	}
}

// NewEKSClient sets the EKS client used when performing the GKE requests.
func (c *EKS) NewEKSClient(*kingpin.ParseContext) error {
	if c.AuthFilename != "" {
	} else if c.AuthFilename = os.Getenv("AWS_APPLICATION_CREDENTIALS"); c.AuthFilename == "" {
		return errors.Errorf("no auth provided set the auth flag or the AWS_APPLICATION_CREDENTIALS env variable")
	}

	cl := eks.New(awsSession.Must(awsSession.NewSession()), &aws.Config{
		Credentials: credentials.NewSharedCredentials(c.AuthFilename, "prombench"),
	})

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
	req := eksCluster{}
	for _, deployment := range c.eksResources {

		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		log.Printf("Cluster create request: name:'%v'", req.Cluster.Name)
		_, err := c.clientEKS.CreateCluster(&req.Cluster)
		if err != nil {
			log.Fatalf("Couldn't create cluster '%v', file:%v ,err: %v", req.Cluster.Name, deployment.FileName, err)
		}

		err = provider.RetryUntilTrue(
			fmt.Sprintf("creating cluster:%v", req.Cluster.Name),
			provider.GlobalRetryCount,
			func() (bool, error) { return c.clusterRunning(*req.Cluster.Name) },
		)

		if err != nil {
			log.Fatalf("creating cluster err:%v", err)
		}

		for _, nodegroupReq := range req.NodeGroups {
			nodegroupReq.ClusterName = req.Cluster.Name
			log.Printf("Nodegroup create request: NodeGroupName: '%v', ClusterName: '%v'", nodegroupReq.NodegroupName, req.Cluster.Name)
			_, err := c.clientEKS.CreateNodegroup(&nodegroupReq)
			if err != nil {
				log.Fatalf("Couldn't create nodegroup '%v' for cluster '%v, file:%v ,err: %v", nodegroupReq.NodegroupName, req.Cluster.Name, deployment.FileName, err)
				break
			}

			err = provider.RetryUntilTrue(
				fmt.Sprintf("creating nodegroup:%v for cluster:%v", nodegroupReq.NodegroupName, req.Cluster.Name),
				provider.GlobalRetryCount,
				func() (bool, error) { return c.nodeGroupCreated(*nodegroupReq.NodegroupName, *req.Cluster.Name) },
			)

			if err != nil {
				log.Fatalf("creating nodegroup err:%v", err)
			}
		}
	}
	return nil
}

// ClusterDelete deletes a eks Cluster
func (c *EKS) ClusterDelete(*kingpin.ParseContext) error {
	req := eksCluster{}
	for _, deployment := range c.eksResources {

		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		// To delete a cluster we have to manually delete all cluster
		log.Printf("Removing all nodepools for '%s'", *req.Cluster.Name)

		// Listing all nodepools for cluster
		reqL := &eks.ListNodegroupsInput{
			ClusterName: req.Cluster.Name,
		}

		for true {
			resL, err := c.clientEKS.ListNodegroups(reqL)
			if err != nil {
				log.Fatalf("listing nodepools err:%v", err)
			}

			for _, nodegroup := range resL.Nodegroups {
				log.Printf("Removing nodepool '%s' in cluster '%s'", *nodegroup, *req.Cluster.Name)

				reqD := eks.DeleteNodegroupInput{
					ClusterName:   req.Cluster.Name,
					NodegroupName: nodegroup,
				}
				_, err := c.clientEKS.DeleteNodegroup(&reqD)
				if err != nil {
					log.Fatalf("Couldn't create nodegroup '%v' for cluster '%v ,err: %v", *nodegroup, req.Cluster.Name, err)
					break
				}

				err = provider.RetryUntilTrue(
					fmt.Sprintf("creating nodegroup:%v for cluster:%v", *nodegroup, *req.Cluster.Name),
					provider.GlobalRetryCount,
					func() (bool, error) { return c.nodeGroupDeleted(*nodegroup, *req.Cluster.Name) },
				)

				if err != nil {
					log.Fatalf("deleting nodegroup err:%v", err)
				}
			}

			if resL.NextToken == nil {
				break
			} else {
				reqL.NextToken = resL.NextToken
			}
		}

		reqD := &eks.DeleteClusterInput{
			Name: req.Cluster.Name,
		}

		log.Printf("Removing cluster '%v'", reqD.Name)

		err := provider.RetryUntilTrue(
			fmt.Sprintf("deleting cluster:%v", reqD.Name),
			provider.GlobalRetryCount,
			func() (bool, error) { return c.clusterDeleted(*reqD.Name) })

		if err != nil {
			log.Fatalf("removing cluster err:%v", err)
		}
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

func (c *EKS) clusterDeleted(name string) (bool, error) {
	req := &eks.DescribeClusterInput{
		Name: aws.String(name),
	}
	clusterRes, err := c.clientEKS.DescribeCluster(req)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == eks.ErrCodeNotFoundException {
			return true, nil
		}
		return false, fmt.Errorf("Couldn't get cluster status: %v", err)
	}

	log.Printf("Cluster '%v' status: %v", name, *clusterRes.Cluster.Status)
	return false, nil
}

// NodeGroupCreate creates a new k8s nodegroup in an existing cluster.
func (c *EKS) NodeGroupCreate(*kingpin.ParseContext) error {
	req := eksCluster{}
	for _, deployment := range c.eksResources {

		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		for _, nodegroupReq := range req.NodeGroups {
			nodegroupReq.ClusterName = req.Cluster.Name
			log.Printf("Nodegroup create request: NodeGroupName: '%v', ClusterName: '%v'", nodegroupReq.NodegroupName, req.Cluster.Name)
			_, err := c.clientEKS.CreateNodegroup(&nodegroupReq)
			if err != nil {
				log.Fatalf("Couldn't create nodegroup '%v' for cluster '%v, file:%v ,err: %v", nodegroupReq.NodegroupName, req.Cluster.Name, deployment.FileName, err)
				break
			}

			err = provider.RetryUntilTrue(
				fmt.Sprintf("creating nodegroup:%v for cluster:%v", nodegroupReq.NodegroupName, req.Cluster.Name),
				provider.GlobalRetryCount,
				func() (bool, error) { return c.nodeGroupCreated(*nodegroupReq.NodegroupName, *req.Cluster.Name) },
			)

			if err != nil {
				log.Fatalf("creating nodegroup err:%v", err)
			}
		}
	}
	return nil
}

// NodeGroupDelete deletes a k8s nodegroup in an existing cluster
func (c *EKS) NodeGroupDelete(*kingpin.ParseContext) error {
	req := eksCluster{}
	for _, deployment := range c.eksResources {
		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		for _, nodegroupReq := range req.NodeGroups {
			nodegroupReq.ClusterName = req.Cluster.Name
			log.Printf("Nodegroup delete request: NodeGroupName: '%v', ClusterName: '%v'", nodegroupReq.NodegroupName, req.Cluster.Name)
			reqD := eks.DeleteNodegroupInput{
				ClusterName:   req.Cluster.Name,
				NodegroupName: nodegroupReq.NodegroupName,
			}
			_, err := c.clientEKS.DeleteNodegroup(&reqD)
			if err != nil {
				log.Fatalf("Couldn't create nodegroup '%v' for cluster '%v, file:%v ,err: %v", nodegroupReq.NodegroupName, req.Cluster.Name, deployment.FileName, err)
				break
			}
			err = provider.RetryUntilTrue(
				fmt.Sprintf("creating nodegroup:%v for cluster:%v", nodegroupReq.NodegroupName, req.Cluster.Name),
				provider.GlobalRetryCount,
				func() (bool, error) { return c.nodeGroupDeleted(*nodegroupReq.NodegroupName, *req.Cluster.Name) },
			)

			if err != nil {
				log.Fatalf("deleting nodegroup err:%v", err)
			}

		}
	}
	return nil
}

func (c *EKS) nodeGroupCreated(nodegroupName, clusterName string) (bool, error) {
	req := &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodegroupName),
	}
	nodegroupRes, err := c.clientEKS.DescribeNodegroup(req)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == eks.ErrCodeNotFoundException {
			return false, nil
		}
		return false, fmt.Errorf("Couldn't get nodegroupname status: %v", err)
	}
	if *nodegroupRes.Nodegroup.Status == eks.NodegroupStatusActive {
		return true, nil
	}

	log.Printf("Nodegroup '%v' for Cluster '%v' status: %v", nodegroupName, clusterName, *nodegroupRes.Nodegroup.Status)
	return false, nil

}

func (c *EKS) nodeGroupDeleted(nodegroupName, clusterName string) (bool, error) {
	req := &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodegroupName),
	}
	nodegroupRes, err := c.clientEKS.DescribeNodegroup(req)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == eks.ErrCodeNotFoundException {
			return true, nil
		}
		return false, fmt.Errorf("Couldn't get nodegroupname status: %v", err)
	}

	log.Printf("Nodegroup '%v' for Cluster '%v' status: %v", nodegroupName, clusterName, *nodegroupRes.Nodegroup.Status)
	return false, nil
}

// AllNodeGroupsRunning returns an error if at least one node pool is not running
func (c *EKS) AllNodeGroupsRunning(*kingpin.ParseContext) error {
	req := eksCluster{}
	for _, deployment := range c.eksResources {
		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}
		for _, nodegroup := range req.NodeGroups {
			isRunning, err := c.nodeGroupCreated(*nodegroup.NodegroupName, *req.Cluster.Name)
			if err != nil {
				log.Fatalf("error fetching nodegroup info")
			}
			if !isRunning {
				log.Fatalf("nodepool not running name: %v", *nodegroup.NodegroupName)
			}
		}
	}
	return nil
}

// AllNodeGroupsDeleted returns an error if at least one node pool is not deleted
func (c *EKS) AllNodeGroupsDeleted(*kingpin.ParseContext) error {
	req := eksCluster{}
	for _, deployment := range c.eksResources {
		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}
		for _, nodegroup := range req.NodeGroups {
			isRunning, err := c.nodeGroupCreated(*nodegroup.NodegroupName, *req.Cluster.Name)
			if err != nil {
				log.Fatalf("error fetching nodegroup info")
			}
			if isRunning {
				log.Fatalf("nodepool not running name: %v", *nodegroup.NodegroupName)
			}
		}
	}
	return nil
}

// NewK8sProvider sets the k8s provider used for deploying k8s manifests
func (c *EKS) NewK8sProvider(*kingpin.ParseContext) error {

	// TODO: Complete this function

	return nil
}

// ResourceApply calls k8s.ResourceApply to apply the k8s objects in the manifest files.
func (c *EKS) ResourceApply(*kingpin.ParseContext) error {
	if err := c.k8sProvider.ResourceApply(c.k8sResources); err != nil {
		log.Fatal("error while applying a resource err:", err)
	}
	return nil
}

// ResourceDelete calls k8s.ResourceDelete to apply the k8s objects in the manifest files.
func (c *EKS) ResourceDelete(*kingpin.ParseContext) error {
	if err := c.k8sProvider.ResourceDelete(c.k8sResources); err != nil {
		log.Fatal("error while deleting objects from a manifest file err:", err)
	}
	return nil
}
