package k8s

import (
	"context"
	"fmt"
	"log"

	"github.com/pkg/errors"
	apiCoreV1 "k8s.io/api/core/v1"
	apiExtensionsV1beta1 "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"

	"strings"

	"github.com/prometheus/prombench/provider"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// K8s is the main provider struct.
type K8s struct {
	clt *kubernetes.Clientset
	ctx context.Context
}

// New returns a k8s client that can apply and delete resources.
func New(ctx context.Context, config clientcmdapi.Config) (*K8s, error) {
	restConfig, err := clientcmd.NewDefaultClientConfig(config, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "config error")
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "client error")
	}

	return &K8s{
		ctx: ctx,
		clt: clientset,
	}, nil
}

// ResourceApply applies manifest files.
// The input map key is the filename and the bytes slice is the actual file content.
// It expect files in the official k8s format.
func (c *K8s) ResourceApply(deployments map[string][]byte) error {
	for name, content := range deployments {
		separator := "---"
		decode := scheme.Codecs.UniversalDeserializer().Decode

		for _, text := range strings.Split(string(content), separator) {
			text = strings.TrimSpace(text)
			if len(text) == 0 {
				continue
			}

			resource, _, err := decode([]byte(text), nil, nil)
			if err != nil {
				return errors.Wrapf(err, "decoding the resource file:%v", name)
			}
			if resource == nil {
				continue
			}

			switch resource.GetObjectKind().GroupVersionKind().Kind {
			case "ClusterRole":
				return c.clusterRoleApply(resource)
			case "ClusterRoleBinding":
				return c.clusterRoleBindingApply(resource)
			case "ConfigMap":
				return c.configMapApply(resource)
			case "DaemonSet":
				return c.daemonSetApply(resource)
			case "Deployment":
				return c.deploymentApply(resource)
			case "Ingress":
				return c.ingressApply(resource)
			case "Namespace":
				return c.nameSpaceApply(resource)
			case "Role":
				return c.roleApply(resource)
			case "RoleBinding":
				return c.roleBindingApply(resource)
			case "Service":
				return c.serviceApply(resource)
			case "ServiceAccount":
				return c.serviceAccountApply(resource)
			}
		}
	}
	return nil
}

// ResourceDelete deletes all resources defined in the resource files.
// The input map key is the filename and the bytes slice is the actual file content.
// It expect files in the official k8s format.
func (c *K8s) ResourceDelete(deployments map[string][]byte) error {
	for name, content := range deployments {
		separator := "---"
		decode := scheme.Codecs.UniversalDeserializer().Decode

		for _, text := range strings.Split(string(content), separator) {
			text = strings.TrimSpace(text)
			if len(text) == 0 {
				continue
			}

			resource, _, err := decode([]byte(text), nil, nil)
			if err != nil {
				return errors.Wrapf(err, "decoding the resource file: %v", name)
			}
			if resource == nil {
				continue
			}
			switch resource.GetObjectKind().GroupVersionKind().Kind {
			case "ClusterRole":
				return c.clusterRoleDelete(resource)
			case "ClusterRoleBinding":
				return c.clusterRoleBindingDelete(resource)
			// Deleting namespace will delete all components in the namespace. Don't need to delete separately.
			case "Namespace":
				return c.namespaceDelete(resource)
			}
		}
	}
	return nil
}

func (c *K8s) clusterRoleApply(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*rbac.ClusterRole)
		client := c.clt.RbacV1().ClusterRoles()
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v ", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		return nil
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}

}

func (c *K8s) clusterRoleBindingApply(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*rbac.ClusterRoleBinding)
		client := c.clt.RbacV1().ClusterRoleBindings()
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v ", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v ", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) configMapApply(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.ConfigMap)
		client := c.clt.CoreV1().ConfigMaps(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) daemonSetApply(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1beta1":
		req := resource.(*apiExtensionsV1beta1.DaemonSet)
		client := c.clt.ExtensionsV1beta1().DaemonSets(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	c.daemonsetReady(resource)
	return nil
}

func (c *K8s) deploymentApply(resource runtime.Object) error {
	req := resource.(*apiExtensionsV1beta1.Deployment)
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1beta1":
		client := c.clt.ExtensionsV1beta1().Deployments(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return provider.RetryUntilTrue(
		fmt.Sprintf("applying deployment:%v", req.Name),
		func() (bool, error) { return c.deploymentReady(resource) })
}

func (c *K8s) ingressApply(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1beta1":
		req := resource.(*apiExtensionsV1beta1.Ingress)
		client := c.clt.ExtensionsV1beta1().Ingresses(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) nameSpaceApply(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.Namespace)
		client := c.clt.CoreV1().Namespaces()
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)

	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) roleApply(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*rbac.Role)
		client := c.clt.RbacV1().Roles(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) roleBindingApply(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*rbac.RoleBinding)
		client := c.clt.RbacV1().RoleBindings(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) serviceAccountApply(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.ServiceAccount)
		client := c.clt.CoreV1().ServiceAccounts(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) serviceApply(resource runtime.Object) error {
	req := resource.(*apiCoreV1.Service)
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.CoreV1().Services(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v", kind)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			}); err != nil {
				return errors.Wrapf(err, "resource update failed - kind: %v", kind)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v", kind)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}

	return provider.RetryUntilTrue(
		fmt.Sprintf("applying service:%v", req.Name),
		func() (bool, error) { return c.serviceExists(resource) })
}

func (c *K8s) clusterRoleDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*rbac.ClusterRole)
		client := c.clt.RbacV1().ClusterRoles()
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			return errors.Wrapf(err, "resource delete failed - kind: %v ", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) clusterRoleBindingDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*rbac.ClusterRoleBinding)
		client := c.clt.RbacV1().ClusterRoleBindings()
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}
func (c *K8s) configMapDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.ConfigMap)
		client := c.clt.CoreV1().ConfigMaps(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) daemonSetDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiExtensionsV1beta1.DaemonSet)
		client := c.clt.ExtensionsV1beta1().DaemonSets(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) deploymentDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiExtensionsV1beta1.Deployment)
		client := c.clt.ExtensionsV1beta1().Deployments(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) ingressDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiExtensionsV1beta1.Ingress)
		client := c.clt.ExtensionsV1beta1().Ingresses(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) namespaceDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.Namespace)
		client := c.clt.CoreV1().Namespaces()
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleting - kind: %v , name: %v", kind, req.Name)
		return provider.RetryUntilTrue(
			fmt.Sprintf("deleting namespace:%v", req.Name),
			func() (bool, error) { return c.namespaceDeleted(resource) })

	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) roleDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*rbac.Role)
		client := c.clt.RbacV1().Roles(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) roleBindingDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*rbac.RoleBinding)
		client := c.clt.RbacV1().RoleBindings(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) serviceDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.Service)
		client := c.clt.CoreV1().Services(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) serviceAccountDelete(resource runtime.Object) error {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.ServiceAccount)
		client := c.clt.CoreV1().ServiceAccounts(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v", kind)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v", v)
	}
	return nil
}

func (c *K8s) serviceExists(resource runtime.Object) (bool, error) {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.Service)
		client := c.clt.CoreV1().Services(req.Namespace)

		res, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "Checking Service resource status failed")
		}
		if res.Spec.Type == apiCoreV1.ServiceTypeLoadBalancer {
			// k8s API currently just supports LoadBalancerStatus
			if len(res.Status.LoadBalancer.Ingress) > 0 {
				log.Printf("\tService %s Details", req.Name)
				for _, x := range res.Status.LoadBalancer.Ingress {
					log.Printf("\t\thttp://%s:%d", x.IP, res.Spec.Ports[0].Port)
				}
				return true, nil
			}
			return false, nil

		}
		return false, fmt.Errorf("unsuported service type:%v", res.Spec.Type)
	default:
		return false, fmt.Errorf("unknown object version: %v", v)
	}
}

func (c *K8s) deploymentReady(resource runtime.Object) (bool, error) {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiExtensionsV1beta1.Deployment)
		client := c.clt.ExtensionsV1beta1().Deployments(req.Namespace)

		res, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "Checking Deployment resource status failed  %v", err)
		}
		if res.Status.UnavailableReplicas == 0 {
			return true, nil
		}
		return false, nil
	default:
		return false, fmt.Errorf("unknown object version: %v", v)
	}
}

func (c *K8s) daemonsetReady(resource runtime.Object) (bool, error) {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiExtensionsV1beta1.DaemonSet)
		client := c.clt.ExtensionsV1beta1().DaemonSets(req.Namespace)

		res, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "Checking DaemonSet resource status failed  %v", err)
		}
		if res.Status.NumberUnavailable == 0 {
			return true, nil
		}
	default:
		return false, fmt.Errorf("unknown object version: %v", v)
	}
	return false, nil
}

func (c *K8s) namespaceDeleted(resource runtime.Object) (bool, error) {
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.Namespace)
		client := c.clt.CoreV1().Namespaces()

		if _, err := client.Get(req.Name, apiMetaV1.GetOptions{}); err != nil {
			if apiErrors.IsNotFound(err) {
				return false, nil
			}
			return false, errors.Wrapf(err, "Couldn't get namespace info:%v", err)
		}
		return true, nil
	default:
		return false, fmt.Errorf("unknown object version: %v", v)
	}
}
