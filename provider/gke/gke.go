package gke

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"text/template"
	"time"

	gke "cloud.google.com/go/container/apiv1"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"gopkg.in/alecthomas/kingpin.v2"
	yamlGo "gopkg.in/yaml.v2"
	apiCoreV1 "k8s.io/api/core/v1"
	apiExtensionsV1beta1 "k8s.io/api/extensions/v1beta1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/retry"
)

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
	// The config file for cluster operations.
	clusterConfig *containerpb.CreateClusterRequest
	// The gke client used when performing GKE requests.
	clientGKE *gke.ClusterManagerClient
	// The k8s client used when performing deployment requests.
	clientset *kubernetes.Clientset
	// Holds the deployments files to apply to the cluster.
	ResourceFiles []string
	// Deployment vaiables to subtitude in the deployment files.
	ResourceVars map[string]string

	ctx context.Context
}

// NewGKEClient sets the GKE client used when performing GKE requests.
func (c *GKE) NewGKEClient(*kingpin.ParseContext) error {
	// See https://cloud.google.com/docs/authentication/.
	// Use GOOGLE_APPLICATION_CREDENTIALS environment variable to specify
	// a service account key file to authenticate to the API.

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
	content, err := ioutil.ReadFile(c.ClusterConfigFile)
	if err != nil {
		log.Fatalf("error reading the config file:%v", err)
	}

	config := &containerpb.CreateClusterRequest{}
	if err = yamlGo.UnmarshalStrict(content, config); err != nil {
		log.Fatalf("error parsing the config file:%v", err)
	}
	c.clusterConfig = config
	return nil
}

// ClusterCreate sreates a new k8s cluster
func (c *GKE) ClusterCreate(*kingpin.ParseContext) error {
	log.Printf("Cluster create request: %+v", c.clusterConfig)

	res, err := c.clientGKE.CreateCluster(c.ctx, c.clusterConfig)
	if err != nil {
		log.Fatalf("Couldn't create a cluster:%v", err)
	}
	log.Printf("Cluster request: %+v", res)

	log.Printf("Cluster %s create is called for project %s and zone %s.", c.clusterConfig.Cluster.Name, c.clusterConfig.ProjectId, c.clusterConfig.Zone)

	return c.waitForCluster()
}

// ClusterDelete deletes a k8s cluster.
func (c *GKE) ClusterDelete(*kingpin.ParseContext) error {

	req := &containerpb.DeleteClusterRequest{
		ProjectId: c.clusterConfig.ProjectId,
		Zone:      c.clusterConfig.Zone,
		ClusterId: c.clusterConfig.Cluster.Name,
	}

	log.Printf("Removing cluster %v from project %v, zone %v", req.ClusterId, req.ProjectId, req.Zone)

	if _, err := c.clientGKE.DeleteCluster(c.ctx, req); err != nil {
		log.Fatal(err)
	}

	log.Printf("Cluster %s set for deletion", req.ClusterId)
	return nil
}

// NewResourceClient sets the client used for deployment requests.
func (c *GKE) NewResourceClient(*kingpin.ParseContext) error {
	req := &containerpb.GetClusterRequest{
		ProjectId: c.clusterConfig.ProjectId,
		Zone:      c.clusterConfig.Zone,
		ClusterId: c.clusterConfig.Cluster.Name,
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

// ResourceApply iterates over all files passed as a cli argument
// and creates or updates the resource definitions to the k8s cluster.
//
// Each file can container more than one resource definition where `apiVersion` is used as separator.
//
// Any deployment variables passed to the cli will be replaced in the manifests files following the golang text template format.
func (c *GKE) ResourceApply(*kingpin.ParseContext) error {

	for _, file := range c.ResourceFiles {
		fileContent, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalf("error while reading the manifest file:%v", err)
		}

		fileContentParsed := bytes.NewBufferString("")

		if err := template.Must(template.New("deployment").Parse(string(fileContent))).Execute(fileContentParsed, c.ResourceVars); err != nil {
			log.Println("executing template:", err)
		}

		splitBy := "apiVersion"
		decode := scheme.Codecs.UniversalDeserializer().Decode

		for k, text := range strings.Split(fileContentParsed.String(), splitBy) {
			if k%2 != 0 { // The even elements return the splitBy string so we don't need those.
				deployment, _, err := decode([]byte(splitBy+text), nil, nil)
				if err != nil {
					log.Fatalf("error while decoding the manifest file: %v", err)
				}
				switch deployment.GetObjectKind().GroupVersionKind().Kind {
				case "Deployment":
					c.deploymentApply(deployment)
				case "ConfigMap":
					c.configMapApply(deployment)
				}

			}
		}
	}

	return nil
}

func (c *GKE) deploymentApply(deployment runtime.Object) {

	switch deployment.GetObjectKind().GroupVersionKind().Version {
	case "v1beta1":
		client := c.clientset.ExtensionsV1beta1().Deployments(apiCoreV1.NamespaceDefault)
		res, err := client.Create(deployment.(*apiExtensionsV1beta1.Deployment))
		fmt.Println(res)
		fmt.Println(err)
	}

}

func (c *GKE) configMapApply(deployment runtime.Object) {
	switch deployment.GetObjectKind().GroupVersionKind().Version {
	case "v1":
		req := deployment.(*apiCoreV1.ConfigMap)
		var res *apiCoreV1.ConfigMap
		client := c.clientset.CoreV1().ConfigMaps(apiCoreV1.NamespaceDefault)

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing server config mapes:%v", err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			// Update Deployment
			//    You have two options to Update() this Deployment:
			//
			//    1. Modify the "deployment" variable and call: Update(deployment).
			//       This works like the "kubectl replace" command and it overwrites/loses changes
			//       made by other clients between you Create() and Update() the object.
			//    2. Modify the "result" returned by Get() and retry Update(result) until
			//       you no longer get a conflict error. This way, you can preserve changes made
			//       by other clients between Create() and Update(). This is implemented below
			//			 using the retry utility package included with client-go. (RECOMMENDED)
			//
			// More Info:
			// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency

			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			})
			if err != nil {
				log.Fatalf("config map update failed: %v", err)
			}
			log.Printf("updated config map:%v", req.Name)
		} else {
			// Create Deployment
			res, err := client.Create(deployment.(*apiCoreV1.ConfigMap))

			if err != nil {
				log.Fatalf("deployment create error: %v", err)
			}
			log.Printf("created config map %q.\n", res.GetObjectMeta().GetName())
		}
		fmt.Println(res)
		fmt.Println(err)
	}
}

// ResourceDelete deletes a k8s resource.
func (c *GKE) ResourceDelete(*kingpin.ParseContext) error {
	// deletePolicy := metav1.DeletePropagationForeground

	// deployment := &appsv1.Deployment{}

	// for _, f := range c.DeploymentFiles {
	// 	file, err := os.Open(f)
	// 	if err != nil {
	// 		log.Fatalf("error reading the manifest file:%v", err)
	// 	}
	// 	if err := yaml.NewYAMLOrJSONDecoder(file, 100).Decode(deployment); err != nil {
	// 		log.Fatalf("error reading the manifest file:%v", err)
	// 	}

	// 	if err := c.clientK8SDeployments.Delete(deployment.Name, &metav1.DeleteOptions{
	// 		PropagationPolicy: &deletePolicy,
	// 	}); err != nil {
	// 		log.Printf("deployment delete error: %v", err)

	// 	} else {
	// 		log.Printf("deleted deployment:%v", deployment.Name)
	// 	}
	// }
	return nil
}

func (c *GKE) waitForCluster() error {
	req := &containerpb.GetClusterRequest{
		ProjectId: c.clusterConfig.ProjectId,
		Zone:      c.clusterConfig.Zone,
		ClusterId: c.clusterConfig.Cluster.Name,
	}
	for {
		cluster, err := c.clientGKE.GetCluster(c.ctx, req)
		if err != nil {
			log.Fatalf("Couldn't get cluster info:%v", err)
		}
		if cluster.Status == containerpb.Cluster_RUNNING {
			log.Printf("Cluster %v is running", cluster.Name)
			return nil
		}
		retry := time.Second * 10
		log.Printf("cluster not ready, current status:%v retrying in %v", cluster.Status, retry)
		time.Sleep(retry)
	}
}

func (c *GKE) waitForNodePool() error {
	// req := &containerpb.GetNodePoolRequest{
	// 	ProjectId: c.ProjectID,
	// 	Zone:      c.Zone,
	// 	ClusterId: c.Name,
	// }
	// for {
	// 	nodepool, err := c.clientGKE.GetNodePool(c.ctx, req)
	// 	if err != nil {
	// 		log.Fatalf("Couldn't get node pool info:%v", err)
	// 	}
	// 	if nodepool.Status == containerpb.NodePool_RUNNING {
	// 		log.Printf("Nodepool %v is running", c.Name)
	// 		return nil
	// 	}
	// 	log.Printf("%v nodepool %v", nodepool.Status, c.Name)
	// 	retry := time.Second * 10
	// 	log.Printf("cluster not ready, retrying in %v", retry)
	// 	time.Sleep(retry)
	// }
	return nil
}
