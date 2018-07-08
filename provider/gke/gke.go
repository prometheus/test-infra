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
	"time"

	gke "cloud.google.com/go/container/apiv1"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"gopkg.in/alecthomas/kingpin.v2"
	yamlGo "gopkg.in/yaml.v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const maxTries = 30

// New is the GKE constructor.
func New() *GKE {
	return &GKE{
		ResourceVars: make(map[string]string),
	}
}

// GKE holds the fields used to generate an API request.
type GKE struct {
	ProjectId string

	Zone string

	ClusterId string
	// The cluster config file location provided to the cli.
	ConfigFile string
	// The auth file used to authenticate the cli.
	AuthFile string
	// The gke client for nodepools operations.
	clusterConfig *containerpb.CreateClusterRequest
	// The gke client for nodepools operations.
	nodePoolConfig []*containerpb.CreateNodePoolRequest
	// The gke client used when performing GKE requests.
	clientGKE *gke.ClusterManagerClient
	// The k8s client used when performing resource requests.
	clientset *kubernetes.Clientset
	// Holds the resources files to apply to the cluster.
	ResourceFiles []string
	// Resource vaiables to subtitude in the resource files.
	ResourceVars map[string]string

	ctx context.Context
}

// NewGKEClient sets the GKE client used when performing GKE requests.
func (c *GKE) NewGKEClient(*kingpin.ParseContext) error {
	if c.AuthFile != "" {
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", c.AuthFile)
	} else if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS") == "" {
		log.Fatal("GOOGLE_APPLICATION_CREDENTIALS env is empty. Please run with -a key.json or run `export GOOGLE_APPLICATION_CREDENTIALS=key.json`")
	}
	client, err := gke.NewClusterManagerClient(context.Background())
	if err != nil {
		log.Fatalf("Could not create the client: %v", err)
	}
	c.clientGKE = client
	c.ctx = context.Background()

	return nil
}

// ConfigParse populates and validates the cluster configuraiton options.
func (c *GKE) ConfigParse(*kingpin.ParseContext) error {

	// Read auth file to get the project id
	var authFile string
	if c.AuthFile != "" {
		authFile = c.AuthFile
	} else {
		authFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}
	content, err := ioutil.ReadFile(authFile)
	if err != nil {
		log.Fatalf("Couldn't read auth file: %v", err)
	}
	d := make(map[string]interface{})
	if err := json.Unmarshal(content, &d); err != nil {
		log.Fatalf("Couldn't parse auth file: %v", err)
	}
	projectId, ok := d["project_id"].(string)
	if !ok {
		log.Fatal("Couldn't get project id from the auth file")
	}
	c.ProjectId = projectId

	// Read config file to get zone and cluster name
	content, err = c.applyTemplateVars(c.ConfigFile)
	if err != nil {
		log.Fatalf("Couldn't apply template to file %s: %v", c.ConfigFile, err)
	}
	config := &containerpb.CreateClusterRequest{}
	if err = yamlGo.UnmarshalStrict(content, config); err != nil {
		log.Fatalf("Error parsing the cluster section in config file %f:%v", c.ConfigFile, err)
	}
	config.ProjectId = c.ProjectId
	c.clusterConfig = config
	c.Zone = config.Zone
	c.ClusterId = config.Cluster.Name

	clusterNodePools := config.Cluster.NodePools[:0]

	for _, pool := range config.Cluster.NodePools {
		if pool.Config.Labels["isolation"] == "prometheus" || pool.Config.Labels["isolation"] == "none" {
			config := &containerpb.CreateNodePoolRequest{
				ProjectId: c.ProjectId,
				Zone:      c.Zone,
				ClusterId: c.ClusterId,
				NodePool:  pool,
			}
			c.nodePoolConfig = append(c.nodePoolConfig, config)
		} else {
			clusterNodePools = append(clusterNodePools, pool)
		}
	}
	c.clusterConfig.Cluster.NodePools = clusterNodePools
	return nil
}

/* Cluster operations */
// ClusterCreate creates a new k8s cluster
func (c *GKE) ClusterCreate(*kingpin.ParseContext) error {

	log.Printf("Cluster %s create is called for project %s and zone %s", c.ClusterId, c.ProjectId, c.Zone)
	_, err := c.clientGKE.CreateCluster(c.ctx, c.clusterConfig)
	if err != nil {
		log.Fatalf("Couldn't create a cluster:%v", err)
	}

	return c.waitForClusterCreation()
}

// ClusterDelete deletes a k8s cluster.
func (c *GKE) ClusterDelete(*kingpin.ParseContext) error {

	req := &containerpb.DeleteClusterRequest{
		ProjectId: c.ProjectId,
		Zone:      c.Zone,
		ClusterId: c.ClusterId,
	}

	log.Printf("Removing cluster %v from project %v, zone %v", req.ClusterId, req.ProjectId, req.Zone)

	if _, err := c.clientGKE.DeleteCluster(c.ctx, req); err != nil {
		if strings.Contains(err.Error(), "code = NotFound") {
			log.Printf("Cluster %s not found.", c.ClusterId)
			return nil
		}
		log.Fatalf("Couldn't delete the cluster:%v", err)
	}

	log.Printf("Cluster %s set for deletion", req.ClusterId)
	return c.waitForClusterDeletion()
}

func (c *GKE) waitForClusterCreation() error {
	req := &containerpb.GetClusterRequest{
		ProjectId: c.ProjectId,
		Zone:      c.Zone,
		ClusterId: c.ClusterId,
	}
	for i := 1; i <= maxTries; i++ {
		cluster, err := c.clientGKE.GetCluster(c.ctx, req)
		if err != nil {
			log.Fatalf("Couldn't get cluster info:%v", err)
		}

		if cluster.Status == containerpb.Cluster_ERROR || cluster.Status == containerpb.Cluster_RECONCILING || cluster.Status == containerpb.Cluster_STOPPING {
			log.Fatalf("Cluster Creation failed: %s", cluster.StatusMessage)
		}
		if cluster.Status == containerpb.Cluster_RUNNING {
			log.Printf("Cluster %v is running", cluster.Name)
			return nil
		}
		retry := time.Second * 10
		log.Printf("Cluster not ready, current status:%v Retrying after %v", cluster.Status, retry)
		time.Sleep(retry)
	}
	log.Fatalf("Cluster %v was not created after trying %d times", c.ClusterId, maxTries)
	return nil
}

func (c *GKE) waitForClusterDeletion() error {
	req := &containerpb.GetClusterRequest{
		ProjectId: c.ProjectId,
		Zone:      c.Zone,
		ClusterId: c.ClusterId,
	}
	for i := 1; i <= maxTries; i++ {
		_, err := c.clientGKE.GetCluster(c.ctx, req)
		if err != nil {
			if strings.Contains(err.Error(), "code = NotFound") {
				log.Printf("Cluster %v not found", c.ClusterId)
				return nil
			}
			log.Fatalf("Couldn't get cluster info:%v", err)
		}

		retry := time.Second * 10
		log.Printf("Cluster is being deleted. Retrying after %v.", retry)
		time.Sleep(retry)
	}
	log.Fatalf("Cluster %v was not deleted after trying %d times", c.ClusterId, maxTries)
	return nil
}

/* Node Pool operations */
// NodePoolCreate creates a new k8s node-pool in an existing cluster
func (c *GKE) NodePoolCreate(*kingpin.ParseContext) error {

	for _, pool := range c.nodePoolConfig {
		log.Printf("Received a NodePool create request: %v", pool)
		var i int
		c.checkNodePoolExists(pool)
		for i = 1; i <= maxTries; i++ {
			_, err := c.clientGKE.CreateNodePool(c.ctx, pool)
			if err != nil {
				if strings.Contains(err.Error(), "Please wait and try again once it is done") {
					retry := time.Second * 20
					log.Printf("NodePool operation is ongoing on the cluster. Retrying after 20 seconds.")
					time.Sleep(retry)
					continue
				}
				log.Fatalf("Couldn't create a node-pool:%v", err)
			}
			c.waitForNodePoolCreation(pool)
			break
		}
		if i > maxTries {
			log.Fatalf("NodePool operation was not free after trying %d times", maxTries)
		}
	}
	return nil
}

// NodePoolCreate deletes a new k8s node-pool in an existing cluster
func (c *GKE) NodePoolDelete(*kingpin.ParseContext) error {

	for _, pool := range c.nodePoolConfig {
		log.Printf("Received a NodePool delete request: %v", pool)
		req := &containerpb.DeleteNodePoolRequest{
			ProjectId:  pool.ProjectId,
			Zone:       pool.Zone,
			ClusterId:  pool.ClusterId,
			NodePoolId: pool.NodePool.Name,
		}

		var i int
		for i = 1; i <= maxTries; i++ {
			_, err := c.clientGKE.DeleteNodePool(c.ctx, req)
			if err != nil {
				if strings.Contains(err.Error(), "Please wait and try again once it is done") {
					retry := time.Second * 20
					log.Printf("NodePool operation is ongoing on the cluster. Retrying after 20 seconds.")
					time.Sleep(retry)
					continue
				} else if strings.Contains(err.Error(), "code = NotFound") {
					log.Printf("NodePool %s has already been deleted.", pool.NodePool.Name)
					break
				}
				log.Fatal("Couldn't delete the node-pool:%v", err)
			}
			log.Printf("Node Pool %s set for deletion", pool.NodePool.Name)
			c.waitForNodePoolDeletion(pool)
			break
		}
		if i > maxTries {
			log.Fatalf("NodePool operation was not free after trying %d times", maxTries)
		}
	}
	return nil
}

func (c *GKE) checkNodePoolExists(n *containerpb.CreateNodePoolRequest) error {
	req := &containerpb.GetNodePoolRequest{
		ProjectId:  n.ProjectId,
		Zone:       n.Zone,
		ClusterId:  n.ClusterId,
		NodePoolId: n.NodePool.Name,
	}
	for i := 1; i <= maxTries; i++ {
		nodePool, err := c.clientGKE.GetNodePool(c.ctx, req)
		if err != nil {
			if strings.Contains(err.Error(), "code = NotFound") {
				return nil
			}
			log.Fatalf("Couldn't check node-pool's existence:%v", err)
		}

		if nodePool.Status == containerpb.NodePool_RUNNING || nodePool.Status == containerpb.NodePool_PROVISIONING {
			log.Fatalf("NodePool %v is already running", nodePool.Name)
		}

		if nodePool.Status == containerpb.NodePool_ERROR || nodePool.Status == containerpb.NodePool_RECONCILING || nodePool.Status == containerpb.NodePool_RUNNING_WITH_ERROR {
			log.Fatalf("NodePool %v is unusable: %v. Retry after deleting Prombench instance using /benchmark delete.", nodePool.Name, nodePool.StatusMessage)
		}

		retry := time.Second * 10
		log.Printf("NodePool %v is being deleted. Waiting for it to be deleted before making new one.", nodePool.Name)
		time.Sleep(retry)
	}
	log.Fatalf("NodePool %v was not deleted after trying %d times", n.NodePool.Name, maxTries)
	return nil
}
func (c *GKE) waitForNodePoolCreation(n *containerpb.CreateNodePoolRequest) {
	req := &containerpb.GetNodePoolRequest{
		ProjectId:  n.ProjectId,
		Zone:       n.Zone,
		ClusterId:  n.ClusterId,
		NodePoolId: n.NodePool.Name,
	}
	for i := 1; i <= maxTries; i++ {
		nodePool, err := c.clientGKE.GetNodePool(c.ctx, req)
		if err != nil {
			if strings.Contains(err.Error(), "code = NotFound") {
				retry := time.Second * 10
				log.Printf("Node Pool %v not ready, retrying in %v", n.NodePool.Name, retry)
				time.Sleep(retry)
				continue
			}
			log.Fatalf("Couldn't get node-pool info:%v", err)
		}

		if nodePool.Status == containerpb.NodePool_ERROR || nodePool.Status == containerpb.NodePool_STOPPING || nodePool.Status == containerpb.NodePool_RECONCILING || nodePool.Status == containerpb.NodePool_RUNNING_WITH_ERROR {
			log.Fatalf("NodePool Creation failed: %s", nodePool.StatusMessage)
		}
		if nodePool.Status == containerpb.NodePool_RUNNING {
			log.Printf("NodePool %v is running", nodePool.Name)
			return
		}
		retry := time.Second * 10
		log.Printf("Node Pool %v not ready, current status:%v retrying in %v", nodePool.Name, nodePool.Status, retry)
		time.Sleep(retry)
	}
	log.Fatalf("NodePool %v was not created after trying %d times", n.NodePool.Name, maxTries)
	return
}

func (c *GKE) waitForNodePoolDeletion(n *containerpb.CreateNodePoolRequest) {
	req := &containerpb.GetNodePoolRequest{
		ProjectId:  n.ProjectId,
		Zone:       n.Zone,
		ClusterId:  n.ClusterId,
		NodePoolId: n.NodePool.Name,
	}
	for i := 1; i <= maxTries; i++ {
		nodePool, err := c.clientGKE.GetNodePool(c.ctx, req)
		if err != nil {
			if strings.Contains(err.Error(), "code = NotFound") {
				return
			}
			log.Fatalf("Couldn't get node-pool info:%v", err)
		}

		retry := time.Second * 10
		log.Printf("NodePool %v is being deleted. Retrying after 10 seconds.", nodePool.Name)
		time.Sleep(retry)
	}
	log.Fatalf("NodePool %v was not deleted after trying %d times", n.NodePool.Name, maxTries)
	return
}

// NewResourceClient sets the client used for resource operations.
func (c *GKE) NewResourceClient(*kingpin.ParseContext) error {

	req := &containerpb.GetClusterRequest{
		ProjectId: c.ProjectId,
		Zone:      c.Zone,
		ClusterId: c.ClusterId,
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

	restConfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		log.Fatalf("clientset error: %v", err)
	}

	c.clientset = clientset
	return nil
}

// ResourceApply iterates over all files passed as cli arguments
// and creates or updates the resource definitions on the k8s cluster.
//
// Each file can contain more than one resource definition where `apiVersion` is used as separator.
//
// Any variables passed to the cli will be replaced in the resources files following the golang text template format.
func (c *GKE) ResourceApply(*kingpin.ParseContext) error {
	fileList := []string{}

	for _, file := range c.ResourceFiles {
		// handle directory
		if info, err := os.Stat(file); err == nil && info.IsDir() {
			err := filepath.Walk(file, func(path string, f os.FileInfo, err error) error {
				if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
					fileList = append(fileList, path)
				}
				return nil
			})
			if err != nil {
				log.Fatalf("error while reading directory%v", err)
			}
		} else {
			fileList = append(fileList, file)
		}
	}

	for _, file := range fileList {

		fileContentParsed, err := c.applyTemplateVars(file)
		if err != nil {
			log.Fatalf("Couldn't apply template to resource file %s: %v", file, err)
		}

		separator := "---"
		decode := scheme.Codecs.UniversalDeserializer().Decode

		for _, text := range strings.Split(string(fileContentParsed), separator) {
			text = strings.TrimSpace(text)
			if len(text) == 0 {
				continue
			}

			resource, _, err := decode([]byte(text), nil, nil)
			if err != nil {
				log.Fatalf("error while decoding the resource file: %v", err)
			}
			if resource == nil {
				continue
			}

			switch resource.GetObjectKind().GroupVersionKind().Kind {
			case "ClusterRole":
				c.clusterRoleApply(resource)
			case "ClusterRoleBinding":
				c.clusterRoleBindingApply(resource)
			case "ConfigMap":
				c.configMapApply(resource)
			case "DaemonSet":
				c.daemonSetApply(resource)
			case "Deployment":
				c.deploymentApply(resource)
			case "Ingress":
				c.ingressApply(resource)
			case "Namespace":
				c.nameSpaceApply(resource)
			case "Role":
				c.roleApply(resource)
			case "RoleBinding":
				c.roleBindingApply(resource)
			case "Service":
				c.serviceApply(resource)
			case "ServiceAccount":
				c.serviceAccountApply(resource)
			}
		}
	}
	return nil
}

// ResourceDelete iterates over all files passed as a cli argument
// and deletes all resources defined in the resource files.
//
// Each file can container more than one resource definition where `apiVersion` is used as separator.
func (c *GKE) ResourceDelete(*kingpin.ParseContext) error {
	fileList := []string{}

	for _, file := range c.ResourceFiles {
		// handle directory
		if info, err := os.Stat(file); err == nil && info.IsDir() {
			err := filepath.Walk(file, func(path string, f os.FileInfo, err error) error {
				if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
					fileList = append(fileList, path)
				}
				return nil
			})
			if err != nil {
				log.Fatalf("error while reading directory%v", err)
			}
		} else {
			fileList = append(fileList, file)
		}
	}

	for _, file := range fileList {

		fileContentParsed, err := c.applyTemplateVars(file)
		if err != nil {
			log.Fatalf("Couldn't apply template to resource file %s: %v", file, err)
		}

		separator := "---"
		decode := scheme.Codecs.UniversalDeserializer().Decode

		for _, text := range strings.Split(string(fileContentParsed), separator) {
			text = strings.TrimSpace(text)
			if len(text) == 0 {
				continue
			}

			resource, _, err := decode([]byte(text), nil, nil)

			if err != nil {
				log.Fatalf("error while decoding the resource file: %v", err)
			}
			if resource == nil {
				continue
			}
			switch resource.GetObjectKind().GroupVersionKind().Kind {
			case "ClusterRole":
				c.clusterRoleDelete(resource)
			case "ClusterRoleBinding":
				c.clusterRoleBindingDelete(resource)
			/* Deleting namespace will delete all components in the namespace. Don't need to delete separately */
			case "Namespace":
				c.nameSpaceDelete(resource)
			}
		}
	}
	return nil
}

func (c *GKE) applyTemplateVars(file string) ([]byte, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("Error reading file %v:%v", file, err)
	}

	fileContentParsed := bytes.NewBufferString("")
	t := template.New("resource").Option("missingkey=error")
	if err := template.Must(t.Parse(string(content))).Execute(fileContentParsed, c.ResourceVars); err != nil {
		log.Fatalf("Failed to execute template:%v", err)
	}
	return fileContentParsed.Bytes(), nil
}
