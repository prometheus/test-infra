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

const maxTries = 25

// New is the GKE constructor.
func New() *GKE {
	return &GKE{
		ResourceVars: make(map[string]string),
	}
}

// GKE holds the fields used to generate an API request.
type GKE struct {
	// The config file location provided to the cli.
	ClusterConfigFile string
	// The auth file used to authenticate the cli.
	AuthFile string
	// The config for nodepools operations.
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

	// Read auth file and get the project id
	var authFile string
	if c.AuthFile != "" {
		authFile = c.AuthFile
	} else {
		authFile = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	content, err := ioutil.ReadFile(authFile)
	if err != nil {
		log.Fatalf("couldn't read auth file: %v", err)
	}
	d := make(map[string]interface{})
	if err := json.Unmarshal(content, &d); err != nil {
		log.Fatalf("couldn't parse auth file: %v", err)
	}
	projectID, ok := d["project_id"].(string)
	if !ok {
		log.Fatal("couldn't get project id from the auth file")
	}

	// creating node-pool requests from config file
	content, err = ioutil.ReadFile(c.ClusterConfigFile)
	if err != nil {
		log.Fatalf("error reading the config file:%v", err)
	}

	splitBy := "zone"

	for _, text := range strings.Split(string(content), splitBy)[1:] {
		config := &containerpb.CreateNodePoolRequest{}
		if err = yamlGo.UnmarshalStrict([]byte(splitBy+text), config); err != nil {
			log.Fatalf("error parsing the config file:%v", err)
		}
		config.ProjectId = projectID
		c.nodePoolConfig = append(c.nodePoolConfig, config)
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

// NodePoolCreate creates a new k8s node-pool in an existing cluster
func (c *GKE) NodePoolCreate(*kingpin.ParseContext) error {

	for _, pool := range c.nodePoolConfig {
		log.Printf("Received a NodePool create request: %+v", pool)

		c.checkNodePoolExists(pool)
		_, err := c.clientGKE.CreateNodePool(c.ctx, pool)
		if err != nil {
			log.Fatalf("Couldn't create a node-pool:%v", err)
		}
		c.waitForNodePoolCreation(pool)
	}
	return nil
}

func (c *GKE) waitForNodePoolCreation(n *containerpb.CreateNodePoolRequest) error {
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
			return nil
		}
		retry := time.Second * 10
		log.Printf("Node Pool %v not ready, current status:%v retrying in %v", nodePool.Name, nodePool.Status, retry)
		time.Sleep(retry)
	}
	log.Fatalf("NodePool %v was not created after trying %d times", n.NodePool.Name, maxTries)
	return nil
}

// NodePoolCreate deletes a new k8s node-pool in an existing cluster
func (c *GKE) NodePoolDelete(*kingpin.ParseContext) error {

	for _, pool := range c.nodePoolConfig {
		req := &containerpb.DeleteNodePoolRequest{
			ProjectId:  pool.ProjectId,
			Zone:       pool.Zone,
			ClusterId:  pool.ClusterId,
			NodePoolId: pool.NodePool.Name,
		}

		if _, err := c.clientGKE.DeleteNodePool(c.ctx, req); err != nil {
			log.Fatal("Couldn't delete the node-pool:%v", err)
		}
		log.Printf("Node Pool %s set for deletion", pool.NodePool.Name)
		c.waitForNodePoolDeletion(pool)
		log.Printf("Removed nodepool %v from cluster %v in project %v, zone %v", pool.NodePool.Name, pool.ClusterId, pool.ProjectId, pool.Zone)
	}
	return nil
}

func (c *GKE) waitForNodePoolDeletion(n *containerpb.CreateNodePoolRequest) error {
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
			log.Fatalf("Couldn't get node-pool info:%v", err)
		}

		retry := time.Second * 10
		log.Printf("NodePool %v is being deleted. Waiting for it to be deleted.", nodePool.Name)
		time.Sleep(retry)
	}
	log.Fatalf("NodePool %v was not deleted after trying %d times", n.NodePool.Name, maxTries)
	return nil
}

// NewResourceClient sets the client used for resource operations.
func (c *GKE) NewResourceClient(*kingpin.ParseContext) error {

	if len(c.nodePoolConfig) == 0 {
		log.Fatalf("NodePool config not availible in config files")
	}

	req := &containerpb.GetClusterRequest{
		ProjectId: c.nodePoolConfig[0].ProjectId,
		Zone:      c.nodePoolConfig[0].Zone,
		ClusterId: c.nodePoolConfig[0].ClusterId,
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
				if filepath.Ext(path) == ".yaml" {
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
		fileContent, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalf("error while reading the resource file:%v", err)
		}

		fileContentParsed := bytes.NewBufferString("")
		t := template.New("resource")
		t.Option("missingkey=error")
		if err := template.Must(t.Parse(string(fileContent))).Execute(fileContentParsed, c.ResourceVars); err != nil {
			log.Fatalf("executing template:%v", err)
		}

		splitBy := "apiVersion"
		decode := scheme.Codecs.UniversalDeserializer().Decode

		for _, text := range strings.Split(fileContentParsed.String(), splitBy)[1:] {
			resource, _, err := decode([]byte(splitBy+text), nil, nil)

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
				if filepath.Ext(path) == ".yaml" {
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
		fileContent, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalf("error while reading the resource file:%v", err)
		}
		fileContentParsed := bytes.NewBufferString("")

		if err := template.Must(template.New("resource").Parse(string(fileContent))).Execute(fileContentParsed, c.ResourceVars); err != nil {
			log.Println("executing template:", err)
		}

		splitBy := "apiVersion"
		decode := scheme.Codecs.UniversalDeserializer().Decode

		for _, text := range strings.Split(fileContentParsed.String(), splitBy)[1:] {
			resource, _, err := decode([]byte(splitBy+text), nil, nil)

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
			case "ConfigMap":
				c.configMapDelete(resource)
			case "DaemonSet":
				c.daemonSetDelete(resource)
			case "Deployment":
				c.deploymentDelete(resource)
			case "Namespace":
				c.nameSpaceDelete(resource)
			case "Role":
				c.roleDelete(resource)
			case "RoleBinding":
				c.roleBindingDelete(resource)
			case "Service":
				c.serviceDelete(resource)
			case "ServiceAccount":
				c.serviceAccountDelete(resource)
			}
		}
	}
	return nil
}
