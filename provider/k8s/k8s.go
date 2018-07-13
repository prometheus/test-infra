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
				err = c.clusterRoleApply(resource)
			case "ClusterRoleBinding":
				err = c.clusterRoleBindingApply(resource)
			case "ConfigMap":
				err = c.configMapApply(resource)
			case "DaemonSet":
				err = c.daemonSetApply(resource)
			case "Deployment":
				err = c.deploymentApply(resource)
			case "Ingress":
				err = c.ingressApply(resource)
			case "Namespace":
				err = c.nameSpaceApply(resource)
			case "Role":
				err = c.roleApply(resource)
			case "RoleBinding":
				err = c.roleBindingApply(resource)
			case "Service":
				err = c.serviceApply(resource)
			case "ServiceAccount":
				err = c.serviceAccountApply(resource)
			}
			if err != nil {
				return errors.Wrapf(err, "apply resources from manifest file:%v", name)
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
			switch kind := strings.ToLower(resource.GetObjectKind().GroupVersionKind().Kind); kind {
			case "clusterrole":
				err = c.clusterRoleDelete(resource)
			case "clusterrolebinding":
				err = c.clusterRoleBindingDelete(resource)
			case "configmap":
				err = c.configMapDelete(resource)
			case "daemonset":
				err = c.daemonsetDelete(resource)
			case "deployment":
				err = c.deploymentDelete(resource)
			case "ingress":
				err = c.ingressDelete(resource)
			case "namespace":
				err = c.namespaceDelete(resource)
			case "service":
				err = c.serviceDelete(resource)
			case "serviceaccount":
				err = c.serviceAccountDelete(resource)
			default:
				err = fmt.Errorf("deleting request for unimplimented resource type:%v", kind)
			}

			if err != nil {
				return errors.Wrapf(err, "delete resources from manifest file:%v", name)
			}
		}
	}
	return nil
}

func (c *K8s) clusterRoleApply(resource runtime.Object) error {
	req := resource.(*rbac.ClusterRole)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.RbacV1().ClusterRoles()

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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		return nil
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}

}

func (c *K8s) clusterRoleBindingApply(resource runtime.Object) error {
	req := resource.(*rbac.ClusterRoleBinding)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.RbacV1().ClusterRoleBindings()
		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) configMapApply(resource runtime.Object) error {
	req := resource.(*apiCoreV1.ConfigMap)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":

		client := c.clt.CoreV1().ConfigMaps(req.Namespace)

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) daemonSetApply(resource runtime.Object) error {
	req := resource.(*apiExtensionsV1beta1.DaemonSet)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1beta1":
		client := c.clt.ExtensionsV1beta1().DaemonSets(req.Namespace)
		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	c.daemonsetReady(resource)
	return nil
}

func (c *K8s) deploymentApply(resource runtime.Object) error {
	req := resource.(*apiExtensionsV1beta1.Deployment)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1beta1":
		client := c.clt.ExtensionsV1beta1().Deployments(req.Namespace)
		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return provider.RetryUntilTrue(
		fmt.Sprintf("applying deployment:%v", req.Name),
		func() (bool, error) { return c.deploymentReady(resource) })
}

func (c *K8s) ingressApply(resource runtime.Object) error {
	req := resource.(*apiExtensionsV1beta1.Ingress)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1beta1":
		client := c.clt.ExtensionsV1beta1().Ingresses(req.Namespace)
		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) nameSpaceApply(resource runtime.Object) error {
	req := resource.(*apiCoreV1.Namespace)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.CoreV1().Namespaces()
		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)

	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) roleApply(resource runtime.Object) error {
	req := resource.(*rbac.Role)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.RbacV1().Roles(req.Namespace)
		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) roleBindingApply(resource runtime.Object) error {
	req := resource.(*rbac.RoleBinding)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.RbacV1().RoleBindings(req.Namespace)
		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) serviceAccountApply(resource runtime.Object) error {
	req := resource.(*apiCoreV1.ServiceAccount)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.CoreV1().ServiceAccounts(req.Namespace)
		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) serviceApply(resource runtime.Object) error {
	req := resource.(*apiCoreV1.Service)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.CoreV1().Services(req.Namespace)
		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "error listing resource : %v, name: %v", kind, req.Name)
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
				return errors.Wrapf(err, "resource update failed - kind: %v, name: %v", kind, req.Name)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
			return nil
		} else if _, err := client.Create(req); err != nil {
			return errors.Wrapf(err, "resource creation failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}

	return provider.RetryUntilTrue(
		fmt.Sprintf("applying service:%v", req.Name),
		func() (bool, error) { return c.serviceExists(resource) })
}

func (c *K8s) clusterRoleDelete(resource runtime.Object) error {
	req := resource.(*rbac.ClusterRole)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.RbacV1().ClusterRoles()
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			return errors.Wrapf(err, "resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) clusterRoleBindingDelete(resource runtime.Object) error {
	req := resource.(*rbac.ClusterRoleBinding)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.RbacV1().ClusterRoleBindings()
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}
func (c *K8s) configMapDelete(resource runtime.Object) error {
	req := resource.(*apiCoreV1.ConfigMap)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.CoreV1().ConfigMaps(req.Namespace)
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) daemonSetDelete(resource runtime.Object) error {
	req := resource.(*apiExtensionsV1beta1.DaemonSet)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.ExtensionsV1beta1().DaemonSets(req.Namespace)
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) deploymentDelete(resource runtime.Object) error {
	req := resource.(*apiExtensionsV1beta1.Deployment)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.ExtensionsV1beta1().Deployments(req.Namespace)
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) ingressDelete(resource runtime.Object) error {
	req := resource.(*apiExtensionsV1beta1.Ingress)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.ExtensionsV1beta1().Ingresses(req.Namespace)
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) namespaceDelete(resource runtime.Object) error {
	req := resource.(*apiCoreV1.Namespace)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.CoreV1().Namespaces()
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleting - kind: %v , name: %v", kind, req.Name)
		return provider.RetryUntilTrue(
			fmt.Sprintf("deleting namespace:%v", req.Name),
			func() (bool, error) { return c.namespaceDeleted(resource) })
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
}

func (c *K8s) roleDelete(resource runtime.Object) error {
	req := resource.(*rbac.Role)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.RbacV1().Roles(req.Namespace)
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) roleBindingDelete(resource runtime.Object) error {
	req := resource.(*rbac.RoleBinding)
	kind := resource.GetObjectKind().GroupVersionKind().Kind
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.RbacV1().RoleBindings(req.Namespace)
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) serviceDelete(resource runtime.Object) error {
	req := resource.(*apiCoreV1.Service)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.CoreV1().Services(req.Namespace)
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) serviceAccountDelete(resource runtime.Object) error {
	req := resource.(*apiCoreV1.ServiceAccount)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		client := c.clt.CoreV1().ServiceAccounts(req.Namespace)
		delPolicy := apiMetaV1.DeletePropagationForeground
		if err := client.Delete(req.Name, &apiMetaV1.DeleteOptions{PropagationPolicy: &delPolicy}); err != nil {
			log.Printf("resource delete failed - kind: %v, name: %v", kind, req.Name)
		}
		log.Printf("resource deleted - kind: %v , name: %v", kind, req.Name)
	default:
		return fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return nil
}

func (c *K8s) serviceExists(resource runtime.Object) (bool, error) {
	req := resource.(*apiCoreV1.Service)
	kind := resource.GetObjectKind().GroupVersionKind().Kind

	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
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
		return false, fmt.Errorf("Checking not implemented for service type:%v name:%v", res.Spec.Type, req.Name)
	default:
		return false, fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
}

func (c *K8s) deploymentReady(resource runtime.Object) (bool, error) {
	req := resource.(*apiExtensionsV1beta1.Deployment)
	kind := resource.GetObjectKind().GroupVersionKind().Kind
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiExtensionsV1beta1.Deployment)
		client := c.clt.ExtensionsV1beta1().Deployments(req.Namespace)

		res, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "Checking Deployment resource:'%v' status failed err:%v", req.Name, err)
		}
		if res.Status.UnavailableReplicas == 0 {
			return true, nil
		}
		return false, nil
	default:
		return false, fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
}

func (c *K8s) daemonsetReady(resource runtime.Object) (bool, error) {
	req := resource.(*apiExtensionsV1beta1.DaemonSet)
	kind := resource.GetObjectKind().GroupVersionKind().Kind
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiExtensionsV1beta1.DaemonSet)
		client := c.clt.ExtensionsV1beta1().DaemonSets(req.Namespace)

		res, err := client.Get(req.Name, apiMetaV1.GetOptions{})
		if err != nil {
			return false, errors.Wrapf(err, "Checking DaemonSet resource:'%v' status failed err:%v", req.Name, err)
		}
		if res.Status.NumberUnavailable == 0 {
			return true, nil
		}
	default:
		return false, fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
	return false, nil
}

func (c *K8s) namespaceDeleted(resource runtime.Object) (bool, error) {
	req := resource.(*apiCoreV1.Namespace)
	kind := resource.GetObjectKind().GroupVersionKind().Kind
	switch v := resource.GetObjectKind().GroupVersionKind().Version; v {
	case "v1":
		req := resource.(*apiCoreV1.Namespace)
		client := c.clt.CoreV1().Namespaces()

		if _, err := client.Get(req.Name, apiMetaV1.GetOptions{}); err != nil {
			if apiErrors.IsNotFound(err) {
				return false, nil
			}
			return false, errors.Wrapf(err, "Couldn't get namespace '%v' err:%v", req.Name, err)
		}
		return true, nil
	default:
		return false, fmt.Errorf("unknown object version: %v kind:'%v', name:'%v'", v, kind, req.Name)
	}
}
