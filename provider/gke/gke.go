package gke

import (
	"context"
	"log"
	"time"

	gke "cloud.google.com/go/container/apiv1"

	containerpb "google.golang.org/genproto/googleapis/container/v1"
	"gopkg.in/alecthomas/kingpin.v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

const (
	cloudScope        = "https://www.googleapis.com/auth/cloud-platform"
	monitorWriteScope = "https://www.googleapis.com/auth/monitoring.write"
	storageReadScope  = "https://www.googleapis.com/auth/devstorage.read_only"
	statusRunning     = "RUNNING"
)

// Cluster holds the fields used to generate an API request.
type Cluster struct {
	// ProjectID is the ID of your project to use when creating a cluster.
	ProjectID string
	// Enable the dashboard.
	Dashboard bool
	// The zone to launch the cluster
	Zone string
	// The number of nodes to create in this cluster
	NodeCount int32
	// The authentication information for accessing the master
	MasterAuth *containerpb.MasterAuth
	// The name of this cluster
	Name     string
	NodePool string
	// Configuration for the Kubernetes Dashboard
	KubernetesDashboard bool
	// The service client used when performing different requests.
	client *gke.ClusterManagerClient
	ctx    context.Context
}

// New sets the GKE client used when authenticating each API request.
func (c *Cluster) New(*kingpin.ParseContext) error {
	// See https://cloud.google.com/docs/authentication/.
	// Use GOOGLE_APPLICATION_CREDENTIALS environment variable to specify
	// a service account key file to authenticate to the API.

	client, err := gke.NewClusterManagerClient(context.Background())
	if err != nil {
		log.Fatalf("Could not create the client: %v", err)
	}
	c.client = client
	c.ctx = context.Background()

	return nil
}

// List current k8s clusters.
func (c *Cluster) List(*kingpin.ParseContext) error {

	req := &containerpb.ListClustersRequest{
		ProjectId: c.ProjectID,
		Zone:      c.Zone,
	}
	list, err := c.client.ListClusters(c.ctx, req)
	if err != nil {
		log.Fatalf("failed to list clusters: %v", err)
	}
	for _, v := range list.Clusters {
		log.Printf("Cluster %q (%s) master_version: v%s", v.Name, v.Status, v.CurrentMasterVersion)
	}
	return nil
}

// Create a new k8s cluster
func (c *Cluster) Create(*kingpin.ParseContext) error {
	req := &containerpb.CreateClusterRequest{
		ProjectId: c.ProjectID,
		Zone:      c.Zone,
		Cluster: &containerpb.Cluster{
			Name:             c.Name,
			InitialNodeCount: c.NodeCount,
			// If unspecified, the defaults are used.
			NodeConfig: &containerpb.NodeConfig{},
			// The authentication information for accessing the master endpoint.
			MasterAuth: &containerpb.MasterAuth{},

			AddonsConfig: &containerpb.AddonsConfig{
				KubernetesDashboard: &containerpb.KubernetesDashboard{
					Disabled: c.Dashboard,
				},
			},

			// ResourceLabels map[string]string `protobuf:"bytes,15,rep,name=resource_labels,json=resourceLabels" json:"resource_labels,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
		},
	}
	log.Printf("Cluster request: %+v", req)

	rep, err := c.client.CreateCluster(c.ctx, req)

	if err != nil {
		log.Fatalf("Couldn't create a cluster:%v", err)
	}
	log.Printf("Cluster %s create is called for project %s and zone %s. Status %v", rep.Name, c.ProjectID, rep.Zone, rep.Status)

	return c.waitForCluster()
}

// Delete a k8s cluster.
func (c *Cluster) Delete(*kingpin.ParseContext) error {
	log.Printf("Removing cluster %v from project %v, zone %v", c.Name, c.ProjectID, c.Zone)

	req := &containerpb.DeleteClusterRequest{
		ProjectId: c.ProjectID,
		Zone:      c.Zone,
		ClusterId: c.Name,
	}

	if _, err := c.client.DeleteCluster(c.ctx, req); err != nil {
		log.Fatal(err)
	}

	log.Printf("Cluster %s set for deletion", c.Name)
	return nil
}

// // Apply a aminfest to the k8s cluster.
// func (c *Cluster) Apply(*kingpin.ParseContext) error {
// 	// create the clientset
// 	clientset, err := kubernetes.NewForConfig(config)
// 	if err != nil {
// 		panic(err.Error())
// 	}
// 	for {
// 		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
// 		if err != nil {
// 			panic(err.Error())
// 		}
// 		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))

// 		// Examples for error handling:
// 		// - Use helper functions like e.g. errors.IsNotFound()
// 		// - And/or cast to StatusError and use its properties like e.g. ErrStatus.Message
// 		namespace := "default"
// 		pod := "example-xxxxx"
// 		_, err = clientset.CoreV1().Pods(namespace).Get(pod, metav1.GetOptions{})
// 		if errors.IsNotFound(err) {
// 			fmt.Printf("Pod %s in namespace %s not found\n", pod, namespace)
// 		} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
// 			fmt.Printf("Error getting pod %s in namespace %s: %v\n",
// 				pod, namespace, statusError.ErrStatus.Message)
// 		} else if err != nil {
// 			panic(err.Error())
// 		} else {
// 			fmt.Printf("Found pod %s in namespace %s\n", pod, namespace)
// 		}

// 		time.Sleep(10 * time.Second)
// 	}
// 	return nil
// }

func (c *Cluster) waitForCluster() error {
	req := &containerpb.GetClusterRequest{
		ProjectId: c.ProjectID,
		Zone:      c.Zone,
		ClusterId: c.Name,
	}
	for {
		cluster, err := c.client.GetCluster(c.ctx, req)
		if err != nil {
			log.Fatalf("Couldn't get cluster info:%v", err)
		}
		if cluster.Status == containerpb.Cluster_RUNNING {
			log.Printf("Cluster %v is running", c.Name)
			return nil
		}
		log.Printf("%v cluster %v", cluster.Status, c.Name)
		retry := time.Second * 10
		log.Printf("cluster not ready, retrying in %v", retry)
		time.Sleep(retry)
	}
}

func (c *Cluster) waitForNodePool() error {
	req := &containerpb.GetNodePoolRequest{
		ProjectId:  c.ProjectID,
		Zone:       c.Zone,
		ClusterId:  c.Name,
		NodePoolId: c.NodePool,
	}
	for {
		nodepool, err := c.client.GetNodePool(c.ctx, req)
		if err != nil {
			log.Fatalf("Couldn't get node pool info:%v", err)
		}
		if nodepool.Status == containerpb.NodePool_RUNNING {
			log.Printf("Nodepool %v is running", c.Name)
			return nil
		}
		log.Printf("%v nodepool %v", nodepool.Status, c.Name)
		retry := time.Second * 10
		log.Printf("cluster not ready, retrying in %v", retry)
		time.Sleep(retry)
	}
}
