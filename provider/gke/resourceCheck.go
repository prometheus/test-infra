package gke

import (
	"log"
	"strings"
	"time"

	apiCoreV1 "k8s.io/api/core/v1"
	apiExtensionsV1beta1 "k8s.io/api/extensions/v1beta1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func (c *GKE) waitForService(resource runtime.Object) {
	req := resource.(*apiCoreV1.Service)
	client := c.clientset.CoreV1().Services(req.Namespace)

	for i := 1; i <= maxTries; i++ {
		res, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			log.Fatalf("Checking Service resource status failed  %v", err)
		}
		if res.Spec.Type == "LoadBalancer" {
			// k8s API currently just supports LoadBalancerStatus
			if len(res.Status.LoadBalancer.Ingress) > 0 {
				log.Printf("\tService %s Details", req.Name)
				for _, x := range res.Status.LoadBalancer.Ingress {
					log.Printf("\t\thttp://%s:%d", x.IP, res.Spec.Ports[0].Port)
				}
				return
			}
			retry := time.Second * 10
			log.Printf("Service %v external IP is being created. Retrying after 10 seconds.", req.Name)
			time.Sleep(retry)
		} else {
			return
		}
	}
	log.Fatalf("Service %v was not created after trying %d times", req.Name, maxTries)
}

func (c *GKE) waitForDeployment(resource runtime.Object) {
	req := resource.(*apiExtensionsV1beta1.Deployment)
	client := c.clientset.ExtensionsV1beta1().Deployments(req.Namespace)

	for i := 1; i <= maxTries; i++ {
		res, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			log.Fatalf("Checking Deployment resource status failed  %v", err)
		}
		if res.Status.UnavailableReplicas == 0 {
			return
		}
		retry := time.Second * 10
		log.Printf("Deployment %v is being created. Retrying after 10 seconds.", req.Name)
		time.Sleep(retry)
	}
	log.Fatalf("Deployment %v was not created after trying %d times", req.Name, maxTries)
}

func (c *GKE) waitForDaemonSet(resource runtime.Object) {
	req := resource.(*apiExtensionsV1beta1.DaemonSet)
	client := c.clientset.ExtensionsV1beta1().DaemonSets(req.Namespace)

	for i := 1; i <= maxTries; i++ {
		res, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			log.Fatalf("Checking DaemonSet resource status failed  %v", err)
		}
		if res.Status.NumberUnavailable == 0 {
			return
		}
		retry := time.Second * 10
		log.Printf("DaemonSet %v is being created. Retrying after 10 seconds.", req.Name)
		time.Sleep(retry)
	}
	log.Fatalf("DaemonSet %v was not created after trying %d times", req.Name, maxTries)
}

func (c *GKE) waitForNameSpaceDeletion(resource runtime.Object) {
	req := resource.(*apiCoreV1.Namespace)
	client := c.clientset.CoreV1().Namespaces()

	for i := 1; i <= maxTries; i++ {
		_, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return
			}
			log.Fatalf("Couldn't get namespace info:%v", err)
		}
		retry := time.Second * 10
		log.Printf("All the components of namespace %v are being deleted. Retrying after 10 seconds.", req.Name)
		time.Sleep(retry)
	}
	log.Fatalf("Namespace %v was not deleted after trying %d times", req.Name, maxTries)
}
