package gke

import (
	"log"
	"time"

	apiCoreV1 "k8s.io/api/core/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func (c *GKE) waitForService(resource runtime.Object) error {
	req := resource.(*apiCoreV1.Service)
	client := c.clientset.CoreV1().Services(apiCoreV1.NamespaceDefault)

	for {
		res, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			log.Fatalf("Checking resource status failed  %v", err)
		}
		if res.Spec.Type == "LoadBalancer" {
			// k8s API currently just supports LoadBalancerStatus
			if len(res.Status.LoadBalancer.Ingress) > 0 {
				log.Printf("\tService %s Details", req.Name)
				for _, x := range res.Status.LoadBalancer.Ingress {
					log.Printf("\t\thttp://%s:%d", x.IP, res.Spec.Ports[0].Port)
				}
				return nil
			}
			retry := time.Second * 10
			log.Printf("Service %v external IP is being created. Waiting for it to be created.", req.Name)
			time.Sleep(retry)
		} else {
			return nil
		}
	}
	return nil
}
