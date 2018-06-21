package gke

import (
	"log"

	apiCoreV1 "k8s.io/api/core/v1"
	apiExtensionsV1beta1 "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func (c *GKE) deploymentDelete(resource runtime.Object) error {
	req := resource.(*apiExtensionsV1beta1.Deployment)
	client := c.clientset.ExtensionsV1beta1().Deployments(req.Namespace)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	delPolicy := apiMetaV1.DeletePropagationForeground
	if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{
		PropagationPolicy: &delPolicy,
	}); err != nil {
		log.Printf("resource delete failed - kind: %v , error: %v", kind, err)

	} else {
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	}
	return nil
}

func (c *GKE) daemonSetDelete(resource runtime.Object) error {
	req := resource.(*apiExtensionsV1beta1.DaemonSet)
	client := c.clientset.ExtensionsV1beta1().DaemonSets(req.Namespace)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	delPolicy := apiMetaV1.DeletePropagationForeground
	if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{
		PropagationPolicy: &delPolicy,
	}); err != nil {
		log.Printf("resource delete failed - kind: %v , error: %v", kind, err)

	} else {
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	}
	return nil
}

func (c *GKE) configMapDelete(resource runtime.Object) error {
	req := resource.(*apiCoreV1.ConfigMap)
	client := c.clientset.CoreV1().ConfigMaps(req.Namespace)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	delPolicy := apiMetaV1.DeletePropagationForeground
	if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{
		PropagationPolicy: &delPolicy,
	}); err != nil {
		log.Printf("resource delete failed - kind: %v , error: %v", kind, err)

	} else {
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	}
	return nil
}

func (c *GKE) nameSpaceDelete(resource runtime.Object) error {
	req := resource.(*apiCoreV1.Namespace)
	client := c.clientset.CoreV1().Namespaces()
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	delPolicy := apiMetaV1.DeletePropagationForeground
	if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{
		PropagationPolicy: &delPolicy,
	}); err != nil {
		log.Printf("resource delete failed - kind: %v , error: %v", kind, err)

	} else {
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	}
	return nil
}

func (c *GKE) serviceDelete(resource runtime.Object) error {
	req := resource.(*apiCoreV1.Service)
	client := c.clientset.CoreV1().Services(req.Namespace)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	delPolicy := apiMetaV1.DeletePropagationForeground
	if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{
		PropagationPolicy: &delPolicy,
	}); err != nil {
		log.Printf("resource delete failed - kind: %v , error: %v", kind, err)

	} else {
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	}
	return nil
}

func (c *GKE) serviceAccountDelete(resource runtime.Object) error {
	req := resource.(*apiCoreV1.ServiceAccount)
	client := c.clientset.CoreV1().ServiceAccounts(req.Namespace)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	delPolicy := apiMetaV1.DeletePropagationForeground
	if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{
		PropagationPolicy: &delPolicy,
	}); err != nil {
		log.Printf("resource delete failed - kind: %v , error: %v", kind, err)

	} else {
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	}
	return nil
}

func (c *GKE) clusterRoleDelete(resource runtime.Object) error {
	req := resource.(*rbac.ClusterRole)
	client := c.clientset.RbacV1().ClusterRoles()
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	delPolicy := apiMetaV1.DeletePropagationForeground
	if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{
		PropagationPolicy: &delPolicy,
	}); err != nil {
		log.Printf("resource delete failed - kind: %v , error: %v", kind, err)

	} else {
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	}
	return nil
}

func (c *GKE) clusterRoleBindingDelete(resource runtime.Object) error {
	req := resource.(*rbac.ClusterRoleBinding)
	client := c.clientset.RbacV1().ClusterRoleBindings()
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	delPolicy := apiMetaV1.DeletePropagationForeground
	if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{
		PropagationPolicy: &delPolicy,
	}); err != nil {
		log.Printf("resource delete failed - kind: %v , error: %v", kind, err)

	} else {
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	}
	return nil
}
