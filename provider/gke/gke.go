package gke

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	gke "google.golang.org/api/container/v1"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	cloudScope        = "https://www.googleapis.com/auth/cloud-platform"
	monitorWriteScope = "https://www.googleapis.com/auth/monitoring.write"
	storageReadScope  = "https://www.googleapis.com/auth/devstorage.read_only"
	statusRunning     = "RUNNING"
)

// Cluster holds the fields used to generate an API request.
type Cluster struct {
	// ProjectID is the ID of your project to use when creating a cluster
	ProjectID string `json:"projectId,omitempty"`
	// The zone to launch the cluster
	Zone string
	// The IP address range of the container pods
	ClusterIpv4Cidr string
	// An optional description of this cluster
	Description string
	// The number of nodes to create in this cluster
	NodeCount int64
	// the kubernetes master version
	MasterVersion string
	// The authentication information for accessing the master
	MasterAuth *gke.MasterAuth
	// the kubernetes node version
	NodeVersion string
	// The name of this cluster
	Name string
	// Parameters used in creating the cluster's nodes
	NodeConfig *gke.NodeConfig
	// Enable alpha feature
	EnableAlphaFeature bool
	// Configuration for the HTTP (L7) load balancing controller addon
	HTTPLoadBalancing bool
	// Configuration for the horizontal pod autoscaling feature, which increases or decreases the number of replica pods a replication controller has based on the resource usage of the existing pods
	HorizontalPodAutoscaling bool
	// Configuration for the Kubernetes Dashboard
	KubernetesDashboard bool
	// Configuration for NetworkPolicy
	NetworkPolicyConfig bool
	// The list of Google Compute Engine locations in which the cluster's nodes should be located
	Locations []string
	// Network
	Network string
	// Sub Network
	SubNetwork string
	// Configuration for LegacyAbac
	LegacyAbac bool
	// NodePool id
	NodePoolID string
	// Image Type
	ImageType string
	// The service client used when performing different requests.
	ServiceClient *gke.Service
}

// SetServiceClient sets a http client and the service account used when authenticating each API request.
func (c *Cluster) New(*kingpin.ParseContext) error {
	// See https://cloud.google.com/docs/authentication/.
	// Use GOOGLE_APPLICATION_CREDENTIALS environment variable to specify
	// a service account key file to authenticate to the API.

	client, err := google.DefaultClient(context.Background(), gke.CloudPlatformScope)
	if err != nil {
		return fmt.Errorf("Could not get authenticated client: %v", err)
	}
	if c.ServiceClient, err = gke.New(client); err != nil {
		return fmt.Errorf("Could not initialize gke client: %v", err)
	}

	return nil
}

// List current k8s clusters.
func (c *Cluster) List(*kingpin.ParseContext) error {

	list, err := c.ServiceClient.Projects.Zones.Clusters.List(c.ProjectID, c.Zone).Do()
	if err != nil {
		return fmt.Errorf("failed to list clusters: %v", err)
	}
	for _, v := range list.Clusters {
		log.Printf("Cluster %q (%s) master_version: v%s", v.Name, v.Status, v.CurrentMasterVersion)

		poolList, err := c.ServiceClient.Projects.Zones.Clusters.NodePools.List(c.ProjectID, c.Zone, v.Name).Do()
		if err != nil {
			return fmt.Errorf("failed to list node pools for cluster %q: %v", v.Name, err)
		}
		for _, np := range poolList.NodePools {
			log.Printf("  -> Pool %q (%s) machineType=%s node_version=v%s autoscaling=%v", np.Name, np.Status,
				np.Config.MachineType, np.Version, np.Autoscaling != nil && np.Autoscaling.Enabled)
		}
	}
	return nil
}

// Create a new k8s cluster
func (c *Cluster) Create(*kingpin.ParseContext) error {

	log.Printf("Cluster request: %+v", c.generateClusterCreateRequest())
	createCall, err := c.ServiceClient.Projects.Zones.Clusters.Create(c.ProjectID, c.Zone, c.generateClusterCreateRequest()).Context(context.Background()).Do()

	log.Printf("Cluster request submitted: %v", c.generateClusterCreateRequest())

	if err != nil && !strings.Contains(err.Error(), "alreadyExists") {
		log.Printf("Contains error %v", err)
		return err
	}
	if err == nil {
		log.Printf("Cluster %s create is called for project %s and zone %s. Status Code %v", c.Name, c.ProjectID, c.Zone, createCall.HTTPStatusCode)
	}

	return c.waitForCluster()
}

// Update an existing k8s cluster.
func (c *Cluster) Update(*kingpin.ParseContext) error {

	log.Printf("Updating cluster. MasterVersion: %s, NodeVersion: %s, NodeCount: %v", c.MasterVersion, c.NodeVersion, c.NodeCount)
	if c.NodePoolID == "" {
		cluster, err := c.ServiceClient.Projects.Zones.Clusters.Get(c.ProjectID, c.Zone, c.Name).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		c.NodePoolID = cluster.NodePools[0].Name
	}

	if c.MasterVersion != "" {
		log.Printf("Updating master to %v version", c.MasterVersion)
		updateCall, err := c.ServiceClient.Projects.Zones.Clusters.Update(c.ProjectID, c.Zone, c.Name, &gke.UpdateClusterRequest{
			Update: &gke.ClusterUpdate{
				DesiredMasterVersion: c.MasterVersion,
			},
		}).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		log.Printf("Cluster %s update is called for project %s and zone %s. Status Code %v", c.Name, c.ProjectID, c.Zone, updateCall.HTTPStatusCode)
		return c.waitForCluster()
	}

	if c.NodeVersion != "" {
		log.Printf("Updating node to %v verison", c.NodeVersion)
		updateCall, err := c.ServiceClient.Projects.Zones.Clusters.NodePools.Update(c.ProjectID, c.Zone, c.Name, c.NodePoolID, &gke.UpdateNodePoolRequest{
			NodeVersion: c.NodeVersion,
		}).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		log.Printf("Nodepool %s update is called for project %s, zone %s and cluster %s. Status Code %v", c.NodePoolID, c.ProjectID, c.Zone, c.Name, updateCall.HTTPStatusCode)
		return c.waitForNodePool()
	}

	if c.NodeCount != 0 {
		log.Printf("Updating node size to %v", c.NodeCount)
		updateCall, err := c.ServiceClient.Projects.Zones.Clusters.NodePools.SetSize(c.ProjectID, c.Zone, c.Name, c.NodePoolID, &gke.SetNodePoolSizeRequest{
			NodeCount: c.NodeCount,
		}).Context(context.Background()).Do()
		if err != nil {
			return err
		}
		log.Printf("Nodepool %s size change is called for project %s, zone %s and cluster %s. Status Code %v", c.NodePoolID, c.ProjectID, c.Zone, c.Name, updateCall.HTTPStatusCode)

		return c.waitForCluster()
	}
	return nil
}

// Delete a k8s cluster.
func (c *Cluster) Delete(*kingpin.ParseContext) error {
	log.Printf("Removing cluster %v from project %v, zone %v", c.Name, c.ProjectID, c.Zone)
	deleteCall, err := c.ServiceClient.Projects.Zones.Clusters.Delete(c.ProjectID, c.Zone, c.Name).Context(context.Background()).Do()
	if err != nil && !strings.Contains(err.Error(), "notFound") {
		return err
	} else if err == nil {
		log.Printf("Cluster %v delete is called. Status Code %v", c.Name, deleteCall.HTTPStatusCode)
	} else {
		log.Printf("Cluster %s doesn't exist", c.Name)
	}
	return nil
}

func (c *Cluster) waitForCluster() error {
	message := ""
	for {
		cluster, err := c.ServiceClient.Projects.Zones.Clusters.Get(c.ProjectID, c.Zone, c.Name).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if cluster.Status == statusRunning {
			log.Printf("Cluster %v is running", c.Name)
			return nil
		}
		if cluster.Status != message {
			log.Printf("%v cluster %v", string(cluster.Status), c.Name)
			message = cluster.Status
		}
		retry := time.Second * 10
		log.Printf("cluster not ready, retrying in %v", retry)
		time.Sleep(retry)
	}
}

func (c *Cluster) waitForNodePool() error {
	message := ""
	for {
		nodepool, err := c.ServiceClient.Projects.Zones.Clusters.NodePools.Get(c.ProjectID, c.Zone, c.Name, c.NodePoolID).Context(context.TODO()).Do()
		if err != nil {
			return err
		}
		if nodepool.Status == statusRunning {
			log.Printf("Nodepool %v is running", c.Name)
			return nil
		}
		if nodepool.Status != message {
			log.Printf("%v nodepool %v", string(nodepool.Status), c.NodePoolID)
			message = nodepool.Status
		}
		retry := time.Second * 10
		log.Printf("cluster not ready, retrying in %v", retry)
		time.Sleep(retry)
	}
}

func (c *Cluster) generateClusterCreateRequest() *gke.CreateClusterRequest {
	request := gke.CreateClusterRequest{
		Cluster: &gke.Cluster{},
	}
	request.Cluster.Name = c.Name
	request.Cluster.Zone = c.Zone
	request.Cluster.InitialClusterVersion = c.MasterVersion
	request.Cluster.InitialNodeCount = c.NodeCount
	request.Cluster.ClusterIpv4Cidr = c.ClusterIpv4Cidr
	request.Cluster.Description = c.Description
	request.Cluster.EnableKubernetesAlpha = c.EnableAlphaFeature
	request.Cluster.AddonsConfig = &gke.AddonsConfig{
		HttpLoadBalancing:        &gke.HttpLoadBalancing{Disabled: !c.HTTPLoadBalancing},
		HorizontalPodAutoscaling: &gke.HorizontalPodAutoscaling{Disabled: !c.HorizontalPodAutoscaling},
		KubernetesDashboard:      &gke.KubernetesDashboard{Disabled: !c.KubernetesDashboard},
		NetworkPolicyConfig:      &gke.NetworkPolicyConfig{Disabled: !c.NetworkPolicyConfig},
	}
	request.Cluster.Network = c.Network
	request.Cluster.Subnetwork = c.SubNetwork
	request.Cluster.LegacyAbac = &gke.LegacyAbac{
		Enabled: c.LegacyAbac,
	}
	request.Cluster.MasterAuth = &gke.MasterAuth{
		Username: "admin",
	}
	request.Cluster.NodeConfig = c.NodeConfig
	return &request
}
