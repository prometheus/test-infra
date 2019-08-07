package gke

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"

	gke "cloud.google.com/go/container/apiv1"
	"github.com/pkg/errors"
	k8sProvider "github.com/prometheus/prombench/pkg/provider/k8s"

	"github.com/prometheus/prombench/pkg/provider"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/alecthomas/kingpin.v2"
	yamlGo "gopkg.in/yaml.v2"

	"google.golang.org/api/option"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// New is the GKE constructor.
func New() *GKE {
	return &GKE{
		DeploymentVars: make(map[string]string),
	}
}

type Resource = provider.Resource

// GKE holds the fields used to generate an API request.
type GKE struct {
	// The auth used to authenticate the cli.
	// Can be a file path or an env variable that includes the json data.
	Auth string
	// file path that includes the json data.
	// TODO Remove when the k8s client supports an auth config option in NewDefaultClientConfig.
	AuthFilePath string
	// The project id for all requests.
	ProjectID string
	// The gke client used when performing GKE requests.
	clientGKE *gke.ClusterManagerClient
	// The k8s provider used when we work with the manifest files.
	k8sProvider *k8sProvider.K8s
	// DeploymentFiles files provided from the cli.
	DeploymentFiles []string
	// Vaiables to subtitude in the DeploymentFiles.
	// These are also used when the command requires some variables that are not provided by the deployment file.
	DeploymentVars map[string]string
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
		log.Fatal("no auth provided! Need to either set the auth flag or the GOOGLE_APPLICATION_CREDENTIALS env variable")
	}

	// Needed by NewK8sProvider.
	c.AuthFilePath = c.Auth

	// When the auth variable points to a file
	// put the file content in the variable.
	if content, err := ioutil.ReadFile(c.Auth); err == nil {
		c.Auth = string(content)
	}

	// Check is auth data is base64 encoded and decode.
	encoded, err := regexp.MatchString("^([A-Za-z0-9+/]{4})*([A-Za-z0-9+/]{3}=|[A-Za-z0-9+/]{2}==)?$", c.Auth)
	if err != nil {
		return err
	}
	if encoded {
		auth, err := base64.StdEncoding.DecodeString(c.Auth)
		if err != nil {
			return err
		}
		c.Auth = string(auth)
	}

	opts := option.WithCredentialsJSON([]byte(c.Auth))

	cl, err := gke.NewClusterManagerClient(context.Background(), opts)
	if err != nil {
		log.Fatalf("Could not create the client: %v", err)
	}
	c.clientGKE = cl
	c.ctx = context.Background()
	return nil
}

// GKEDeploymentsParse parses the cluster/nodepool deployment files and saves the result as bytes grouped by the filename.
// Any variables passed to the cli will be replaced in the resources files following the golang text template format.
func (c *GKE) GKEDeploymentsParse(*kingpin.ParseContext) error {
	c.setProjectID()

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
	c.setProjectID()

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

// setProjectID either from the cli arg or read it from the auth data.
func (c *GKE) setProjectID() {
	if v, ok := c.DeploymentVars["PROJECT_ID"]; !ok || v == "" {
		d := make(map[string]interface{})
		if err := json.Unmarshal([]byte(c.Auth), &d); err != nil {
			log.Fatalf("Couldn't parse auth file: %v", err)
		}
		v, ok := d["project_id"].(string)
		if !ok {
			log.Fatal("Couldn't get project id from the auth file")
		}
		c.DeploymentVars["PROJECT_ID"] = v
	}
}

// ClusterCreate create a new cluster or applies changes to an existing cluster.
func (c *GKE) ClusterCreate(*kingpin.ParseContext) error {
	req := &containerpb.CreateClusterRequest{}
	for _, deployment := range c.gkeResources {

		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %s:%v", deployment.FileName, err)
		}

		log.Printf("Cluster create request: name:'%v', project `%s`,zone `%s`", req.Cluster.Name, req.ProjectId, req.Zone)
		_, err := c.clientGKE.CreateCluster(c.ctx, req)
		if err != nil {
			log.Fatalf("Couldn't create cluster '%v', file:%v ,err: %v", req.Cluster.Name, deployment.FileName, err)
		}

		err = provider.RetryUntilTrue(
			fmt.Sprintf("creating cluster:%v", req.Cluster.Name),
			provider.GlobalRetryCount,
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
			ProjectId: reqC.ProjectId,
			Zone:      reqC.Zone,
			ClusterId: reqC.Cluster.Name,
		}
		log.Printf("Removing cluster '%v', project '%v', zone '%v'", reqD.ClusterId, reqD.ProjectId, reqD.Zone)

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
			return false, fmt.Errorf("unknown reply status error %v", err)
		}
		if st.Code() == codes.NotFound {
			return true, nil
		}
		if st.Code() == codes.FailedPrecondition {
			log.Printf("Cluster in 'FailedPrecondition' state '%s'", err)
			return false, nil
		}
		return false, errors.Wrapf(err, "deleting cluster:%v", req.ClusterId)
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
		return false, fmt.Errorf("Couldn't get cluster status:%v", err)
	}
	if cluster.Status == containerpb.Cluster_ERROR ||
		cluster.Status == containerpb.Cluster_STATUS_UNSPECIFIED ||
		cluster.Status == containerpb.Cluster_STOPPING {
		return false, fmt.Errorf("Cluster not in a status to become ready - %s", cluster.Status)
	}
	if cluster.Status == containerpb.Cluster_RUNNING {
		return true, nil
	}
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
				ProjectId: reqC.ProjectId,
				Zone:      reqC.Zone,
				ClusterId: reqC.Cluster.Name,
				NodePool:  node,
			}
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
			return false, fmt.Errorf("unknown reply status error %v", err)
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
				ProjectId:  reqC.ProjectId,
				Zone:       reqC.Zone,
				ClusterId:  reqC.Cluster.Name,
				NodePoolId: node.Name,
			}
			log.Printf("Removing cluster node pool: `%v`,  cluster '%v', project '%v', zone '%v'", reqD.NodePoolId, reqD.ClusterId, reqD.ProjectId, reqD.Zone)

			err := provider.RetryUntilTrue(
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
			return false, fmt.Errorf("unknown reply status error %v", err)
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

// nodePoolRunning checks whether a nodepool has been created.
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
		return false, fmt.Errorf("Couldn't get node pool status:%v", err)
	}
	if rep.Status == containerpb.NodePool_RUNNING {
		return true, nil
	}

	if rep.Status == containerpb.NodePool_ERROR ||
		rep.Status == containerpb.NodePool_RUNNING_WITH_ERROR ||
		rep.Status == containerpb.NodePool_STOPPING ||
		rep.Status == containerpb.NodePool_STATUS_UNSPECIFIED {
		log.Fatalf("NodePool %s not in a status to become ready: %v", rep.Name, rep.StatusMessage)
	}

	log.Printf("Current cluster node pool '%v' status:%v , %v", rep.Name, rep.Status, rep.StatusMessage)
	return false, nil
}

// NewK8sProvider sets the k8s provider used for deploying k8s manifests.
func (c *GKE) NewK8sProvider(*kingpin.ParseContext) error {
	projectID, ok := c.DeploymentVars["PROJECT_ID"]
	if !ok {
		return fmt.Errorf("missing required PROJECT_ID variable")
	}
	zone, ok := c.DeploymentVars["ZONE"]
	if !ok {
		return fmt.Errorf("missing required ZONE variable")
	}
	clusterID, ok := c.DeploymentVars["CLUSTER_NAME"]
	if !ok {
		return fmt.Errorf("missing required CLUSTER_NAME variable")
	}

	// The k8s client looks for `GOOGLE_APPLICATION_CREDENTIALS`.
	// It is the only way to set the auth for now.
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", c.AuthFilePath)

	// Get the authentication certificate for the cluster using the GKE client.
	req := &containerpb.GetClusterRequest{
		ProjectId: projectID,
		Zone:      zone,
		ClusterId: clusterID,
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
	config.Contexts[rep.Zone] = context
	config.AuthInfos[rep.Zone] = authInfo
	config.CurrentContext = rep.Zone

	c.k8sProvider, err = k8sProvider.New(c.ctx, config)
	if err != nil {
		log.Fatal("k8s provider error", err)
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
