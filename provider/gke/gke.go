package gke

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	gke "cloud.google.com/go/container/apiv1"
	"github.com/pkg/errors"
	k8sProvider "github.com/prometheus/prombench/provider/k8s"

	"github.com/prometheus/prombench/provider"
	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/alecthomas/kingpin.v2"
	yamlGo "gopkg.in/yaml.v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// New is the GKE constructor.
func New() *GKE {
	return &GKE{
		DeploymentVars: make(map[string]string),
	}
}

// GKE holds the fields used to generate an API request.
type GKE struct {
	// The auth file used to authenticate the cli.
	AuthFile string
	// The gke client used when performing GKE requests.
	clientGKE *gke.ClusterManagerClient
	// The k8s provider used when we work with the manifest files.
	k8sProvider *k8sProvider.K8s
	// DeploymentFiles files provided from the cli.
	DeploymentFiles []string
	// Vaiables to subtitude in the DeploymentFiles.
	// These are also used when the command requires some variables that are not provided by the deployment file.
	DeploymentVars map[string]string
	// DeploymentFile content after substituting the variables filename is used as the map key.
	deploymentsContent []provider.ResourceFile

	ctx context.Context
}

// NewGKEClient sets the GKE client used when performing GKE requests.
func (c *GKE) NewGKEClient(*kingpin.ParseContext) error {
	// Set the auth env variable needed to the gke client.
	if c.AuthFile != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", c.AuthFile)
	} else if c.AuthFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); c.AuthFile == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS env is empty. Please run with -a key.json or run `export GOOGLE_APPLICATION_CREDENTIALS=key.json`")
	}
	cl, err := gke.NewClusterManagerClient(context.Background())
	if err != nil {
		log.Fatalf("Could not create the client: %v", err)
	}
	c.clientGKE = cl
	c.ctx = context.Background()
	return nil
}

// DeploymentsParse parses the deployment files and saves the result as bytes grouped by the filename.
// Any variables passed to the cli will be replaced in the resources files following the golang text template format.
func (c *GKE) DeploymentsParse(*kingpin.ParseContext) error {
	var fileList []string
	for _, name := range c.DeploymentFiles {
		if file, err := os.Stat(name); err == nil && file.IsDir() {
			if err := filepath.Walk(name, func(path string, f os.FileInfo, err error) error {
				if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
					fileList = append(fileList, path)
				}
				return nil
			}); err != nil {
				return fmt.Errorf("error reading directory: %v", err)
			}
		} else {
			fileList = append(fileList, name)
		}
	}

	for _, name := range fileList {
		content, err := c.applyTemplateVars(name)
		if err != nil {
			return fmt.Errorf("couldn't apply template to file %s: %v", name, err)
		}
		c.deploymentsContent = append(c.deploymentsContent, provider.ResourceFile{name, content})
	}
	return nil
}

// ClusterCreate create a new cluster or applyes changes to an existing cluster.
func (c *GKE) ClusterCreate(*kingpin.ParseContext) error {
	req := &containerpb.CreateClusterRequest{}
	for _, deployment := range c.deploymentsContent {

		if err := yamlGo.UnmarshalStrict(deployment.Content, req); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %f:%v", deployment.Name, err)
		}

		log.Printf("Cluster create request: name:'%v', project `%s`,zone `%s`", req.Cluster.Name, req.ProjectId, req.Zone)
		_, err := c.clientGKE.CreateCluster(c.ctx, req)
		if err != nil {
			log.Fatalf("Couldn't create cluster '%v', file:%v ,err: %v", deployment.Name, req.Cluster.Name, err)
		}

		err = provider.RetryUntilTrue(
			fmt.Sprintf("creating cluster:%v", req.Cluster.Name),
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
	for _, deployment := range c.deploymentsContent {
		if err := yamlGo.UnmarshalStrict(deployment.Content, reqC); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %f:%v", deployment.Name, err)
		}
		reqD := &containerpb.DeleteClusterRequest{
			ProjectId: reqC.ProjectId,
			Zone:      reqC.Zone,
			ClusterId: reqC.Cluster.Name,
		}
		log.Printf("Removing cluster '%v', project '%v', zone '%v'", reqD.ClusterId, reqD.ProjectId, reqD.Zone)

		err := provider.RetryUntilTrue(
			fmt.Sprintf("deleting cluster:%v", reqD.ClusterId),
			func() (bool, error) { return c.clusterDeleted(reqD) })

		if err != nil {
			log.Fatalf("removing cluster err:%v", err)
		}
	}
	return nil
}

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

// NodePoolCreate creates a new k8s node-pool in an existing cluster
func (c *GKE) NodePoolCreate(*kingpin.ParseContext) error {
	reqC := &containerpb.CreateClusterRequest{}

	for _, deployment := range c.deploymentsContent {
		if err := yamlGo.UnmarshalStrict(deployment.Content, reqC); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %f:%v", deployment.Name, err)
		}

		for _, node := range reqC.Cluster.NodePools {
			reqN := &containerpb.CreateNodePoolRequest{
				ProjectId: reqC.ProjectId,
				Zone:      reqC.Zone,
				ClusterId: reqC.Cluster.Name,
				NodePool:  node,
			}
			log.Printf("Cluster nodepool create request: cluster '%v', nodepool '%v' , project `%s`,zone `%s`", reqN.ClusterId, reqN.NodePool.Name, reqN.ProjectId, reqN.Zone)
			_, err := c.clientGKE.CreateNodePool(c.ctx, reqN)
			if err != nil {
				log.Fatalf("Couldn't create cluster nodepool '%v', file:%v ,err: %v", reqN.NodePool.Name, deployment.Name, err)
			}

			err = provider.RetryUntilTrue(
				fmt.Sprintf("creating nodepool:%v", reqN.NodePool.Name),
				func() (bool, error) {
					return c.nodePoolRunning(reqN.Zone, reqN.ProjectId, reqN.ClusterId, reqN.NodePool.Name)
				})

			if err != nil {
				log.Fatalf("nodepool create err:%v", err)
			}
		}
	}
	return nil
}

// NodePoolDelete deletes a new k8s node-pool in an existing cluster
func (c *GKE) NodePoolDelete(*kingpin.ParseContext) error {
	// Use CreateNodePoolRequest struct to pass the UnmarshalStrict validation and
	// than use the result to create the DeleteNodePoolRequest
	reqC := &containerpb.CreateClusterRequest{}
	for _, deployment := range c.deploymentsContent {

		if err := yamlGo.UnmarshalStrict(deployment.Content, reqC); err != nil {
			log.Fatalf("Error parsing the cluster deployment file %f:%v", deployment.Name, err)
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
				func() (bool, error) { return c.nodePoolDeleted(reqD) })

			if err != nil {
				log.Fatalf("nodepool delete err:%v", err)
			}
		}
	}
	return nil
}

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
			log.Printf("Cluster in 'FailedPrecondition' state '%s'", err)
			return false, nil
		}
		return false, errors.Wrapf(err, "delete cluster node pool:%v", req.NodePoolId)
	}
	log.Printf("cluster node pool status: `%v`", rep.Status)
	return false, nil
}

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
		log.Fatalf("NodePool not in a status to become ready: %v", rep.Name, rep.StatusMessage)
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

	k8s, err := k8sProvider.New(c.ctx, *config)
	if err != nil {
		log.Fatal("k8s provider error", err)
	}
	c.k8sProvider = k8s
	return nil
}

// ResourceApply iterates over all manifest files
// and applies the resource definitions on the k8s cluster.
//
// Each file can contain more than one resource definition where `----` is used as separator.
func (c *GKE) ResourceApply(*kingpin.ParseContext) error {
	if err := c.k8sProvider.ResourceApply(c.deploymentsContent); err != nil {
		log.Fatal("error while applying a resource err:", err)
	}
	return nil
}

// ResourceDelete iterates over all files passed as a cli argument
// and deletes all resources defined in the resource files.
//
// Each file can container more than one resource definition where `---` is used as separator.
func (c *GKE) ResourceDelete(*kingpin.ParseContext) error {
	if err := c.k8sProvider.ResourceDelete(c.deploymentsContent); err != nil {
		log.Fatal("error while deleting objects from a manifest file err:", err)
	}
	return nil
}

func (c *GKE) applyTemplateVars(file string) ([]byte, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("Error reading file %v:%v", file, err)
	}

	// When the PROJECT_ID is not provided from the cli read it from the auth file.
	if v, ok := c.DeploymentVars["PROJECT_ID"]; !ok || v == "" {
		content, err := ioutil.ReadFile(c.AuthFile)
		if err != nil {
			log.Fatalf("Couldn't read auth file: %v", err)
		}
		d := make(map[string]interface{})
		if err := json.Unmarshal(content, &d); err != nil {
			log.Fatalf("Couldn't parse auth file: %v", err)
		}
		v, ok := d["project_id"].(string)
		if !ok {
			log.Fatal("Couldn't get project id from the auth file")
		}
		c.DeploymentVars["PROJECT_ID"] = v
	}

	fileContentParsed := bytes.NewBufferString("")
	t := template.New("resource").Option("missingkey=error")
	// k8s objects can't have dots(.) se we add a custom function to allow normalising the variable values.
	t = t.Funcs(template.FuncMap{
		"normalise": func(t string) string {
			return strings.Replace(t, ".", "-", -1)
		},
	})
	if err := template.Must(t.Parse(string(content))).Execute(fileContentParsed, c.DeploymentVars); err != nil {
		log.Fatalf("Failed to execute parse file: err:%v", file, err)
	}
	return fileContentParsed.Bytes(), nil
}
