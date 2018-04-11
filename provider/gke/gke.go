package gke

import (
	"context"
	"io/ioutil"
	"log"
	"os"
	"time"

	gke "cloud.google.com/go/container/apiv1"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"gopkg.in/alecthomas/kingpin.v2"
	yamlGo "gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/typed/apps/v1beta1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/util/retry"
)

// GKE holds the fields used to generate an API request.
type GKE struct {
	// The location of the config file used to parse all other options.
	ConfigFile string
	// The authentication information for accessing the k8s master endpoint.
	MasterAuth *containerpb.MasterAuth
	// The gke client used when performing GKE requests.
	clientGKE *gke.ClusterManagerClient

	// The k8s client used when performing deployment requests.
	clientK8SDeployments v1beta1.DeploymentInterface
	// Holds the deployments files to apply to the cluster.
	DeploymentFiles []string
	// Deployment vaiables to subtitude in the deployment files.
	DeploymentVars map[string]string

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

// configParse populates and validates the cluster configuraiton options.
func (c *GKE) configParse() *containerpb.CreateClusterRequest {
	content, err := ioutil.ReadFile(c.ConfigFile)
	if err != nil {
		log.Fatalf("error reading the config file:%v", err)
	}

	req := &containerpb.CreateClusterRequest{}
	if err = yamlGo.UnmarshalStrict(content, req); err != nil {
		log.Fatalf("error parsing the config file:%v", err)
	}
	return req
}

// ClusterList lists current k8s clusters.
func (c *GKE) ClusterList(*kingpin.ParseContext) error {

	// req := &containerpb.ListClustersRequest{
	// 	ProjectId: c.ClusterConfig.ProjectID,
	// 	Zone:      c.ClusterConfig.Zone,
	// }
	// list, err := c.clientGKE.ListClusters(c.ctx, req)
	// if err != nil {
	// 	log.Fatalf("failed to list clusters: %v", err)
	// }
	// for _, v := range list.Clusters {
	// 	log.Printf("Cluster %q (%s) master_version: v%s", v.Name, v.Status, v.CurrentMasterVersion)
	// }
	return nil
}

// ClusterGet details for a k8s clusters.
func (c *GKE) ClusterGet(*kingpin.ParseContext) error {

	// req := &containerpb.GetClusterRequest{
	// 	ProjectId: c.ProjectID,
	// 	Zone:      c.Zone,
	// 	ClusterId: c.Name,
	// }
	// rep, err := c.clientGKE.GetCluster(c.ctx, req)
	// if err != nil {
	// 	log.Fatalf("failed to get cluster details: %v", err)
	// }

	// fmt.Printf("%+v", rep)
	return nil
}

// ClusterCreate sreates a new k8s cluster
func (c *GKE) ClusterCreate(*kingpin.ParseContext) error {
	req := c.configParse()
	log.Printf("Cluster create request: %+v", req)

	res, err := c.clientGKE.CreateCluster(c.ctx, req)
	if err != nil {
		log.Fatalf("Couldn't create a cluster:%v", err)
	}
	log.Printf("Cluster request: %+v", res)

	log.Printf("Cluster %s create is called for project %s and zone %s.", req.Cluster.Name, req.ProjectId, req.Zone)

	return c.waitForCluster(req.ProjectId, req.Zone, req.Cluster.Name)
}

// ClusterDelete deletes a k8s cluster.
func (c *GKE) ClusterDelete(*kingpin.ParseContext) error {
	config := c.configParse()

	req := &containerpb.DeleteClusterRequest{
		ProjectId: config.ProjectId,
		Zone:      config.Zone,
		ClusterId: config.Cluster.Name,
	}

	log.Printf("Removing cluster %v from project %v, zone %v", req.ClusterId, req.ProjectId, req.Zone)

	if _, err := c.clientGKE.DeleteCluster(c.ctx, req); err != nil {
		log.Fatal(err)
	}

	log.Printf("Cluster %s set for deletion", req.ClusterId)
	return nil
}

// NewDeploymentClient sets the client used for deployment requests.
func (c *GKE) NewDeploymentClient(*kingpin.ParseContext) error {
	// req := &containerpb.GetClusterRequest{
	// 	ProjectId: c.ProjectID,
	// 	Zone:      c.Zone,
	// 	ClusterId: c.Name,
	// }
	// rep, err := c.clientGKE.GetCluster(c.ctx, req)
	// if err != nil {
	// 	log.Fatalf("failed to get cluster details: %v", err)
	// }

	// // The master auth retrieved from GCP it is base64 encoded so it must be decoded first.
	// caCert, err := base64.StdEncoding.DecodeString(rep.MasterAuth.GetClusterCaCertificate())
	// if err != nil {
	// 	log.Fatalf("failed to decode certificate: %v", err.Error())
	// }

	// cluster := clientcmdapi.NewCluster()
	// cluster.CertificateAuthorityData = []byte(caCert)
	// cluster.Server = fmt.Sprintf("https://%v", rep.Endpoint)

	// context := clientcmdapi.NewContext()
	// context.Cluster = rep.Name
	// context.AuthInfo = rep.Zone

	// authInfo := clientcmdapi.NewAuthInfo()
	// authInfo.AuthProvider = &clientcmdapi.AuthProviderConfig{
	// 	Name: "gcp",
	// 	Config: map[string]string{
	// 		"cmd-args":   "config config-helper --format=json",
	// 		"expiry-key": "{.credential.token_expiry}",
	// 		"token-key":  "{.credential.access_token}",
	// 	},
	// }

	// config := clientcmdapi.NewConfig()
	// config.Clusters[rep.Name] = cluster
	// config.Contexts[rep.Zone] = context
	// config.AuthInfos[rep.Zone] = authInfo
	// config.CurrentContext = rep.Zone

	// restConfig, err := clientcmd.NewDefaultClientConfig(*config, &clientcmd.ConfigOverrides{}).ClientConfig()
	// if err != nil {
	// 	log.Fatalf("config error: %v", err)
	// }
	// clientset, err := kubernetes.NewForConfig(restConfig)
	// if err != nil {
	// 	log.Fatalf("clientset error: %v", err)
	// }

	// c.clientK8SDeployments = clientset.AppsV1beta1().Deployments(apiv1.NamespaceDefault)
	return nil
}

// DeploymentApply applies manifest files to the k8s cluster.
func (c *GKE) DeploymentApply(*kingpin.ParseContext) error {
	deployment := &appsv1.Deployment{}

	for _, f := range c.DeploymentFiles {
		file, err := os.Open(f)
		if err != nil {
			log.Fatalf("error reading the manifest file:%v", err)
		}
		if err := yaml.NewYAMLOrJSONDecoder(file, 100).Decode(deployment); err != nil {
			log.Fatalf("error reading the manifest file:%v", err)
		}

		list, err := c.clientK8SDeployments.List(metav1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing server deployments:%v", err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == deployment.Name {
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

			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, updateErr := c.clientK8SDeployments.Update(deployment)
				return updateErr
			})
			if retryErr != nil {
				log.Fatalf("deployment update failed: %v", retryErr)
			}
			log.Printf("updated deployment:%v", deployment.Name)
		} else {
			// Create Deployment
			result, err := c.clientK8SDeployments.Create(deployment)
			if err != nil {
				log.Fatalf("deployment create error: %v", err)
			}
			log.Printf("created deployment %q.\n", result.GetObjectMeta().GetName())
		}

	}

	return nil
}

// DeploymentDelete deletes a k8s deployment.
func (c *GKE) DeploymentDelete(*kingpin.ParseContext) error {
	deletePolicy := metav1.DeletePropagationForeground

	deployment := &appsv1.Deployment{}

	for _, f := range c.DeploymentFiles {
		file, err := os.Open(f)
		if err != nil {
			log.Fatalf("error reading the manifest file:%v", err)
		}
		if err := yaml.NewYAMLOrJSONDecoder(file, 100).Decode(deployment); err != nil {
			log.Fatalf("error reading the manifest file:%v", err)
		}

		if err := c.clientK8SDeployments.Delete(deployment.Name, &metav1.DeleteOptions{
			PropagationPolicy: &deletePolicy,
		}); err != nil {
			log.Printf("deployment delete error: %v", err)

		} else {
			log.Printf("deleted deployment:%v", deployment.Name)
		}
	}
	return nil
}

func (c *GKE) waitForCluster(project, zone, cluster string) error {
	req := &containerpb.GetClusterRequest{
		ProjectId: project,
		Zone:      zone,
		ClusterId: cluster,
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
