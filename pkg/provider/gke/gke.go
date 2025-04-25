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

package gke

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gke "cloud.google.com/go/container/apiv1"
	"cloud.google.com/go/container/apiv1/containerpb"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/alecthomas/kingpin.v2"
	yamlGo "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	_ "k8s.io/cloud-provider-gcp/pkg/clientauthplugin/gcp"

	"github.com/prometheus/test-infra/pkg/provider"
	k8sProvider "github.com/prometheus/test-infra/pkg/provider/k8s"
)

// New is the GKE constructor.
func New(dr *provider.DeploymentResource) *GKE {
	return &GKE{
		DeploymentResource: dr,
	}
}

type Resource = provider.Resource

// GKE holds the fields used to generate an API request.
type GKE struct {
	// The auth used to authenticate the cli.
	// Can be a file path or an env variable that includes the json data.
	Auth string
	// The project id for all requests.
	ProjectID string
	// The gke client used when performing GKE requests.
	clientGKE *gke.ClusterManagerClient
	// The k8s provider used when we work with the manifest files.
	k8sProvider *k8sProvider.K8s
	// Final DeploymentFiles files.
	DeploymentFiles []string
	// Final DeploymentVars.
	DeploymentVars map[string]string
	// DeployResource to construct DeploymentVars and DeploymentFiles
	DeploymentResource *provider.DeploymentResource
	// Content bytes after parsing the template variables, grouped by filename.
	gkeResources []Resource
	// K8s resource.runtime objects after parsing the template variables, grouped by filename.
	k8sResources []k8sProvider.Resource

	ctx context.Context
}

// NewGKEClient sets the GKE client used when performing GKE requests.
func (c *GKE) NewGKEClient(*kingpin.ParseContext) error {
	// Set the auth env variable needed to the gke client.
	if c.Auth != "" {
	} else if c.Auth = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); c.Auth == "" {
		return fmt.Errorf("no auth provided! Need to either set the auth flag or the GOOGLE_APPLICATION_CREDENTIALS env variable")
	}

	// When the auth variable points to a file
	// put the file content in the variable.
	if content, err := os.ReadFile(c.Auth); err == nil {
		c.Auth = string(content)
	}

	// Check if auth data is base64 encoded and decode it.
	encoded, err := regexp.MatchString("^([A-Za-z0-9+/]{4})*([A-Za-z0-9+/]{3}=|[A-Za-z0-9+/]{2}==)?$", c.Auth)
	if err != nil {
		return err
	}
	if encoded {
		auth, err := base64.StdEncoding.DecodeString(c.Auth)
		if err != nil {
			return fmt.Errorf("could not decode auth data: %w", err)
		}
		c.Auth = string(auth)
	}

	// Create temporary file to store the credentials.
	saFile, err := os.CreateTemp("", "service-account")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	defer saFile.Close()
	if _, err := saFile.Write([]byte(c.Auth)); err != nil {
		return fmt.Errorf("could not write to temp file: %w", err)
	}
	// Set the auth env variable needed to the k8s client.
	// The client looks for this special variable name and it is the only way to set the auth for now.
	// TODO: Remove when the client supports an auth config option in NewDefaultClientConfig.
	// https://github.com/kubernetes/kubernetes/pull/80303
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", saFile.Name())

	opts := option.WithCredentialsJSON([]byte(c.Auth))

	cl, err := gke.NewClusterManagerClient(context.Background(), opts)
	if err != nil {
		return fmt.Errorf("could not create the gke client: %w", err)
	}
	c.clientGKE = cl
	c.ctx = context.Background()

	return nil
}

// SetupDeploymentResources Sets up DeploymentVars and DeploymentFiles
func (c *GKE) SetupDeploymentResources(*kingpin.ParseContext) error {
	c.DeploymentFiles = c.DeploymentResource.DeploymentFiles
	c.DeploymentVars = provider.MergeDeploymentVars(
		c.DeploymentResource.DefaultDeploymentVars,
		c.DeploymentResource.FlagDeploymentVars,
	)
	return nil
}

// GKEDeploymentsParse parses the cluster/nodepool deployment files and saves the result as bytes grouped by the filename.
// Any variables passed to the cli will be replaced in the resources files following the golang text template format.
func (c *GKE) GKEDeploymentsParse(*kingpin.ParseContext) error {
	if err := c.checkDeploymentVarsAndFiles(); err != nil {
		return err
	}

	deploymentResource, err := provider.DeploymentsParse(c.DeploymentFiles, c.DeploymentVars)
	if err != nil {
		log.Fatalf("Couldn't parse deployment files: %v", err)
	}

	c.gkeResources = deploymentResource
	return nil
}

// K8SDeploymentsParse parses the k8s objects deployment files and saves the result as k8s objects grouped by the filename.
// Any variables passed to the cli will be replaced in the resources files following the golang text template format.
func (c *GKE) K8SDeploymentsParse(*kingpin.ParseContext) error {
	if err := c.checkDeploymentVarsAndFiles(); err != nil {
		return err
	}

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
				return fmt.Errorf("decoding the resource file:%v, section:%v...: %w", deployment.FileName, text[:100], err)
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

// checkDeploymentVarsAndFiles checks whether the requied deployment vars are passed.
func (c *GKE) checkDeploymentVarsAndFiles() error {
	reqDepVars := []string{"GKE_PROJECT_ID", "ZONE", "CLUSTER_NAME"}
	for _, k := range reqDepVars {
		if v, ok := c.DeploymentVars[k]; !ok || v == "" {
			return fmt.Errorf("missing required %v variable", k)
		}
	}
	if len(c.DeploymentFiles) == 0 {
		return fmt.Errorf("missing deployment file(s)")
	}
	return nil
}

// ClusterCreate create a new cluster or applies changes to an existing cluster.
func (c *GKE) ClusterCreate(*kingpin.ParseContext) error {
	req := &containerpb.CreateClusterRequest{}
	for _, deployment := range c.gkeResources {
		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		//nolint:staticcheck // SA1019 - Ignore "Do not use.".
		log.Printf("Cluster create request: name:'%v', project `%s`,zone `%s`", req.Cluster.Name, req.ProjectId, req.Zone)
		_, err := c.clientGKE.CreateCluster(c.ctx, req)
		if err != nil {
			log.Fatalf("Couldn't create cluster '%v', file:%v ,err: %v", req.Cluster.Name, deployment.FileName, err)
		}

		err = provider.RetryUntilTrue(
			fmt.Sprintf("creating cluster:%v", req.Cluster.Name),
			provider.GlobalRetryCount,
			//nolint:staticcheck // SA1019 - Ignore "Do not use.".
			func() (bool, error) { return c.clusterRunning(req.Zone, req.ProjectId, req.Cluster.Name) })
		if err != nil {
			log.Fatalf("creating cluster err:%v", err)
		}
	}
	return nil
}

// ClusterDelete deletes a k8s cluster.
func (c *GKE) ClusterDelete(*kingpin.ParseContext) error {
	// Use CreateClusterRequest struct to pass the UnmarshalStrict validation and
	// than use the result to create the DeleteClusterRequest
	reqC := &containerpb.CreateClusterRequest{}
	for _, deployment := range c.gkeResources {
		if err := yamlGo.UnmarshalStrict(deployment.Content, reqC); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}
		reqD := &containerpb.DeleteClusterRequest{
			//nolint:staticcheck // SA1019 - Ignore "Do not use.".
			ProjectId: reqC.ProjectId,
			//nolint:staticcheck // SA1019 - Ignore "Do not use.".
			Zone: reqC.Zone,
			//nolint:staticcheck // SA1019 - Ignore "Do not use.".
			ClusterId: reqC.Cluster.Name,
		}
		//nolint:staticcheck // SA1019 - Ignore "Do not use.".
		log.Printf("Removing cluster '%v', project '%v', zone '%v'", reqD.ClusterId, reqD.ProjectId, reqD.Zone)

		//nolint:staticcheck // SA1019 - Ignore "Do not use.".
		err := provider.RetryUntilTrue(
			fmt.Sprintf("deleting cluster:%v", reqD.ClusterId),
			provider.GlobalRetryCount,
			func() (bool, error) { return c.clusterDeleted(reqD) })
		if err != nil {
			log.Fatalf("removing cluster err:%v", err)
		}
	}
	return nil
}

// clusterDeleted checks whether a cluster has been deleted.
func (c *GKE) clusterDeleted(req *containerpb.DeleteClusterRequest) (bool, error) {
	rep, err := c.clientGKE.DeleteCluster(c.ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return false, fmt.Errorf("unknown reply status error, %w", err)
		}
		if st.Code() == codes.NotFound {
			return true, nil
		}
		if st.Code() == codes.FailedPrecondition {
			log.Printf("Cluster in 'FailedPrecondition' state '%s'", err)
			return false, nil
		}
		//nolint:staticcheck // SA1019 - Ignore "Do not use.".
		return false, fmt.Errorf("deleting cluster:%v: %w", req.ClusterId, err)
	}
	log.Printf("cluster status: `%v`", rep.Status)
	return false, nil
}

// clusterRunning checks whether a cluster is in a running state.
func (c *GKE) clusterRunning(zone, projectID, clusterID string) (bool, error) {
	req := &containerpb.GetClusterRequest{
		ProjectId: projectID,
		Zone:      zone,
		ClusterId: clusterID,
	}
	cluster, err := c.clientGKE.GetCluster(c.ctx, req)
	if err != nil {
		// We don't consider none existing cluster error a failure. So don't return an error here.
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return false, nil
		}
		return false, fmt.Errorf("Couldn't get cluster status: %w", err)
	}
	if cluster.Status == containerpb.Cluster_ERROR ||
		cluster.Status == containerpb.Cluster_STATUS_UNSPECIFIED ||
		cluster.Status == containerpb.Cluster_STOPPING {
		return false, fmt.Errorf("Cluster not in a status to become ready - %s", cluster.Status)
	}
	if cluster.Status == containerpb.Cluster_RUNNING {
		return true, nil
	}
	//nolint:staticcheck // SA1019 - Ignore "Do not use.".
	log.Printf("Cluster '%v' status:%v , %v", projectID, cluster.Status, cluster.StatusMessage)
	return false, nil
}

// NodePoolCreate creates a new k8s node-pool in an existing cluster.
func (c *GKE) NodePoolCreate(*kingpin.ParseContext) error {
	reqC := &containerpb.CreateClusterRequest{}

	for _, deployment := range c.gkeResources {
		if err := yamlGo.UnmarshalStrict(deployment.Content, reqC); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		for _, node := range reqC.Cluster.NodePools {
			reqN := &containerpb.CreateNodePoolRequest{
				//nolint:staticcheck // SA1019 - Ignore "Do not use.".
				ProjectId: reqC.ProjectId,
				//nolint:staticcheck // SA1019 - Ignore "Do not use.".
				Zone: reqC.Zone,
				//nolint:staticcheck // SA1019 - Ignore "Do not use.".
				ClusterId: reqC.Cluster.Name,
				NodePool:  node,
			}
			//nolint:staticcheck // SA1019 - Ignore "Do not use.".
			log.Printf("Cluster nodepool create request: cluster '%v', nodepool '%v' , project `%s`,zone `%s`", reqN.ClusterId, reqN.NodePool.Name, reqN.ProjectId, reqN.Zone)

			err := provider.RetryUntilTrue(
				fmt.Sprintf("nodepool creation:%v", reqN.NodePool.Name),
				provider.GlobalRetryCount,
				func() (bool, error) {
					return c.nodePoolCreated(reqN)
				})
			if err != nil {
				log.Fatalf("Couldn't create cluster nodepool '%v', file:%v ,err: %v", node.Name, deployment.FileName, err)
			}

			err = provider.RetryUntilTrue(
				fmt.Sprintf("checking nodepool running status for:%v", reqN.NodePool.Name),
				provider.GlobalRetryCount,
				func() (bool, error) {
					//nolint:staticcheck // SA1019 - Ignore "Do not use.".
					return c.nodePoolRunning(reqN.Zone, reqN.ProjectId, reqN.ClusterId, reqN.NodePool.Name)
				})
			if err != nil {
				log.Fatalf("Couldn't create cluster nodepool '%v', file:%v ,err: %v", node.Name, deployment.FileName, err)
			}
		}
	}
	return nil
}

// nodePoolCreated checks if there is any ongoing NodePool operation on the cluster
// when creating a NodePool.
func (c *GKE) nodePoolCreated(req *containerpb.CreateNodePoolRequest) (bool, error) {
	rep, err := c.clientGKE.CreateNodePool(c.ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return false, fmt.Errorf("unknown reply status error: %w", err)
		}
		if st.Code() == codes.FailedPrecondition {
			// GKE cannot have two simultaneous nodepool operations running on it
			// Waiting for any ongoing operation to complete before starting new one
			log.Printf("Cluster in 'FailedPrecondition' state '%s'", err)

			return false, nil
		}
		return false, err
	}
	log.Printf("cluster node pool status: `%v`", rep.Status)
	return true, nil
}

// NodePoolDelete deletes a new k8s node-pool in an existing cluster.
func (c *GKE) NodePoolDelete(*kingpin.ParseContext) error {
	// Use CreateNodePoolRequest struct to pass the UnmarshalStrict validation and
	// than use the result to create the DeleteNodePoolRequest
	reqC := &containerpb.CreateClusterRequest{}
	for _, deployment := range c.gkeResources {
		if err := yamlGo.UnmarshalStrict(deployment.Content, reqC); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		for _, node := range reqC.Cluster.NodePools {
			reqD := &containerpb.DeleteNodePoolRequest{
				//nolint:staticcheck // SA1019 - Ignore "Do not use.".
				ProjectId: reqC.ProjectId,
				//nolint:staticcheck // SA1019 - Ignore "Do not use.".
				Zone:       reqC.Zone,
				ClusterId:  reqC.Cluster.Name,
				NodePoolId: node.Name,
			}
			//nolint:staticcheck // SA1019 - Ignore "Do not use.".
			log.Printf("Removing cluster node pool: `%v`,  cluster '%v', project '%v', zone '%v'", reqD.NodePoolId, reqD.ClusterId, reqD.ProjectId, reqD.Zone)

			err := provider.RetryUntilTrue(
				//nolint:staticcheck // SA1019 - Ignore "Do not use.".
				fmt.Sprintf("deleting nodepool:%v", reqD.NodePoolId),
				provider.GlobalRetryCount,
				func() (bool, error) { return c.nodePoolDeleted(reqD) })
			if err != nil {
				log.Fatalf("Couldn't delete cluster nodepool '%v', file:%v ,err: %v", node.Name, deployment.FileName, err)
			}
		}
	}
	return nil
}

// nodePoolDeleted checks whether a nodepool has been deleted.
func (c *GKE) nodePoolDeleted(req *containerpb.DeleteNodePoolRequest) (bool, error) {
	rep, err := c.clientGKE.DeleteNodePool(c.ctx, req)
	if err != nil {
		st, ok := status.FromError(err)
		if !ok {
			return false, fmt.Errorf("unknown reply status error: %w", err)
		}
		if st.Code() == codes.NotFound {
			return true, nil
		}
		if st.Code() == codes.FailedPrecondition {
			// GKE cannot have two simultaneous nodepool operations running on it
			// Waiting for any ongoing operation to complete before starting new one
			log.Printf("Cluster in 'FailedPrecondition' state '%s'", err)

			return false, nil
		}
		return false, err
	}
	log.Printf("cluster node pool status: `%v`", rep.Status)
	return false, nil
}

// nodePoolRunning checks whether a nodepool has been created and is running.
func (c *GKE) nodePoolRunning(zone, projectID, clusterID, poolName string) (bool, error) {
	req := &containerpb.GetNodePoolRequest{
		ProjectId:  projectID,
		Zone:       zone,
		ClusterId:  clusterID,
		NodePoolId: poolName,
	}
	rep, err := c.clientGKE.GetNodePool(c.ctx, req)
	if err != nil {
		// We don't consider none existing cluster node pool a failure. So don't return an error here.
		if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
			return false, nil
		}
		return false, fmt.Errorf("Couldn't get node pool status: %w", err)
	}
	if rep.Status == containerpb.NodePool_RUNNING {
		return true, nil
	}

	if rep.Status == containerpb.NodePool_ERROR ||
		rep.Status == containerpb.NodePool_RUNNING_WITH_ERROR ||
		rep.Status == containerpb.NodePool_STOPPING ||
		rep.Status == containerpb.NodePool_STATUS_UNSPECIFIED {
		//nolint:staticcheck // SA1019 - Ignore "Do not use.".
		log.Fatalf("NodePool %s not in a status to become ready: %v", rep.Name, rep.StatusMessage)
	}

	//nolint:staticcheck // SA1019 - Ignore "Do not use.".
	log.Printf("Current cluster node pool '%v' status:%v , %v", rep.Name, rep.Status, rep.StatusMessage)
	return false, nil
}

// AllNodepoolsRunning returns an error if at least one node pool is not running.
func (c *GKE) AllNodepoolsRunning(*kingpin.ParseContext) error {
	reqC := &containerpb.CreateClusterRequest{}

	for _, deployment := range c.gkeResources {
		if err := yamlGo.UnmarshalStrict(deployment.Content, reqC); err != nil {
			return fmt.Errorf("error parsing the cluster deployment file %s: %w", deployment.FileName, err)
		}

		for _, node := range reqC.Cluster.NodePools {
			//nolint:staticcheck // SA1019 - Ignore "Do not use.".
			isRunning, err := c.nodePoolRunning(reqC.Zone, reqC.ProjectId, reqC.Cluster.Name, node.Name)
			if err != nil {
				log.Fatalf("error fetching nodePool info")
			}
			if !isRunning {
				log.Fatalf("nodepool not running name: %v", node.Name)
			}
		}
	}

	return nil
}

// AllNodepoolsDeleted returns an error if at least one nodepool is not deleted.
func (c *GKE) AllNodepoolsDeleted(*kingpin.ParseContext) error {
	reqC := &containerpb.CreateClusterRequest{}

	for _, deployment := range c.gkeResources {
		if err := yamlGo.UnmarshalStrict(deployment.Content, reqC); err != nil {
			return fmt.Errorf("error parsing the cluster deployment file %s: %w", deployment.FileName, err)
		}

		for _, node := range reqC.Cluster.NodePools {
			//nolint:staticcheck // SA1019 - Ignore "Do not use.".
			isRunning, err := c.nodePoolRunning(reqC.Zone, reqC.ProjectId, reqC.Cluster.Name, node.Name)
			if err != nil {
				log.Fatalf("error fetching nodePool info")
			}
			if isRunning {
				log.Fatalf("nodepool running name: %v", node.Name)
			}
		}
	}

	return nil
}

// NewK8sProvider sets the k8s provider used for deploying k8s manifests.
func (c *GKE) NewK8sProvider(*kingpin.ParseContext) error {
	// Get the authentication certificate for the cluster using the GKE client.
	req := &containerpb.GetClusterRequest{
		ProjectId: c.DeploymentVars["GKE_PROJECT_ID"],
		Zone:      c.DeploymentVars["ZONE"],
		ClusterId: c.DeploymentVars["CLUSTER_NAME"],
	}
	rep, err := c.clientGKE.GetCluster(c.ctx, req)
	if err != nil {
		log.Fatalf("failed to get cluster details: %v", err)
	}

	// The master auth retrieved from GCP it is base64 encoded so it must be decoded first.
	caCert, err := base64.StdEncoding.DecodeString(rep.MasterAuth.GetClusterCaCertificate())
	if err != nil {
		log.Fatalf("failed to decode certificate: %v", err.Error())
	}

	cluster := clientcmdapi.NewCluster()
	cluster.CertificateAuthorityData = []byte(caCert)
	cluster.Server = fmt.Sprintf("https://%v", rep.Endpoint)

	context := clientcmdapi.NewContext()
	context.Cluster = rep.Name
	//nolint:staticcheck // SA1019 - Ignore "Do not use.".
	context.AuthInfo = rep.Zone

	authInfo := clientcmdapi.NewAuthInfo()
	authInfo.AuthProvider = &clientcmdapi.AuthProviderConfig{
		Name: "gcp",
		Config: map[string]string{
			"cmd-args":   "config config-helper --format=json",
			"expiry-key": "{.credential.token_expiry}",
			"token-key":  "{.credential.access_token}",
		},
	}

	config := clientcmdapi.NewConfig()
	config.Clusters[rep.Name] = cluster
	//nolint:staticcheck // SA1019 - Ignore "Do not use.".
	config.Contexts[rep.Zone] = context
	//nolint:staticcheck // SA1019 - Ignore "Do not use.".
	config.AuthInfos[rep.Zone] = authInfo
	//nolint:staticcheck // SA1019 - Ignore "Do not use.".
	config.CurrentContext = rep.Zone

	c.k8sProvider, err = k8sProvider.New(c.ctx, config)
	if err != nil {
		log.Fatal("k8s provider error", err)
	}
	return nil
}

// The CreateNamespace function is used to create the PR namespace and copy the
// blocksync-config and bucket-secret from the default namespace to the prombench-${PR_NUMBER} namespace.
// Block-sync uses these resources to download data from object storage.
// For more information, refer to this PR: https://github.com/prometheus/test-infra/pull/840

func (c *GKE) CreateNamespace(*kingpin.ParseContext) error {
	sourceNS := "default"
	targetNS := "prombench-" + c.DeploymentVars["PR_NUMBER"]
	configMapName := "blocksync-config"
	secretName := "bucket-secret"

	// check if namespace exists
	_, err := c.k8sProvider.Clt.CoreV1().Namespaces().Get(context.TODO(), targetNS, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: targetNS,
				},
			}
			_, err = c.k8sProvider.Clt.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("error creating namespace: %w", err)
			}
		} else {
			return fmt.Errorf("error checking namespace: %w", err)
		}
	}

	// copy ConfigMap
	_, err = c.k8sProvider.Clt.CoreV1().ConfigMaps(targetNS).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		cm, err := c.k8sProvider.Clt.CoreV1().ConfigMaps(sourceNS).Get(context.TODO(), configMapName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting configmap: %w", err)
		}
		cm.ResourceVersion = ""
		cm.Namespace = targetNS
		_, err = c.k8sProvider.Clt.CoreV1().ConfigMaps(targetNS).Create(context.TODO(), cm, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating configmap: %w", err)
		}
	}

	// copy Secret
	_, err = c.k8sProvider.Clt.CoreV1().Secrets(targetNS).Get(context.TODO(), secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		secret, err := c.k8sProvider.Clt.CoreV1().Secrets(sourceNS).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting secret: %w", err)
		}
		secret.ResourceVersion = ""
		secret.Namespace = targetNS
		_, err = c.k8sProvider.Clt.CoreV1().Secrets(targetNS).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating secret in target NS: %w", err)
		}
	}

	return nil
}

// ResourceApply calls k8s.ResourceApply to apply the k8s objects in the manifest files.
func (c *GKE) ResourceApply(*kingpin.ParseContext) error {
	if err := c.k8sProvider.ResourceApply(c.k8sResources); err != nil {
		log.Fatal("error while applying a resource err:", err)
	}
	return nil
}

// ResourceDelete calls k8s.ResourceDelete to apply the k8s objects in the manifest files.
func (c *GKE) ResourceDelete(*kingpin.ParseContext) error {
	if err := c.k8sProvider.ResourceDelete(c.k8sResources); err != nil {
		log.Fatal("error while deleting objects from a manifest file err:", err)
	}
	return nil
}

// GetDeploymentVars shows deployment variables.
func (c *GKE) GetDeploymentVars(_ *kingpin.ParseContext) error {
	fmt.Print("-------------------\n   DeploymentVars   \n------------------- \n")
	for key, value := range c.DeploymentVars {
		fmt.Println(key, " : ", value)
	}

	return nil
}
