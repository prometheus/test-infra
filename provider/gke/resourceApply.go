package gke

import (
	"log"

	apiCoreV1 "k8s.io/api/core/v1"
	apiExtensionsV1beta1 "k8s.io/api/extensions/v1beta1"
	rbac "k8s.io/api/rbac/v1"
	apiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/util/retry"
)

func (c *GKE) deploymentApply(resource runtime.Object) {
	switch resource.GetObjectKind().GroupVersionKind().Version {
	case "v1beta1":
		req := resource.(*apiExtensionsV1beta1.Deployment)
		client := c.clientset.ExtensionsV1beta1().Deployments(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing resource : %v ; error: config maps:%v", kind, err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			})
			if err != nil {
				log.Fatalf("resource update failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else {
			_, err := client.Create(req)

			if err != nil {
				log.Fatalf("resource creation failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		}
	}
}

func (c *GKE) daemonSetApply(resource runtime.Object) {

	switch resource.GetObjectKind().GroupVersionKind().Version {
	case "v1beta1":
		req := resource.(*apiExtensionsV1beta1.DaemonSet)
		client := c.clientset.ExtensionsV1beta1().DaemonSets(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing resource : %v ; error: config maps:%v", kind, err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			})
			if err != nil {
				log.Fatalf("resource update failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else {
			_, err := client.Create(req)

			if err != nil {
				log.Fatalf("resource creation failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		}
	}
}

func (c *GKE) configMapApply(resource runtime.Object) {
	switch resource.GetObjectKind().GroupVersionKind().Version {
	case "v1":
		req := resource.(*apiCoreV1.ConfigMap)
		client := c.clientset.CoreV1().ConfigMaps(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing resource : %v ; error: config maps:%v", kind, err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			})
			if err != nil {
				log.Fatalf("resource update failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else {
			_, err := client.Create(req)

			if err != nil {
				log.Fatalf("resource creation failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		}
	}
}

func (c *GKE) nameSpaceApply(resource runtime.Object) {
	switch resource.GetObjectKind().GroupVersionKind().Version {
	case "v1":
		req := resource.(*apiCoreV1.Namespace)
		client := c.clientset.CoreV1().Namespaces()
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing resource : %v ; error: config maps:%v", kind, err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			})
			if err != nil {
				log.Fatalf("resource update failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else {
			_, err := client.Create(req)
			if err != nil {
				log.Fatalf("resource creation failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		}
		c.waitForService(resource)
	}
}

func (c *GKE) serviceApply(resource runtime.Object) {
	switch resource.GetObjectKind().GroupVersionKind().Version {
	case "v1":
		req := resource.(*apiCoreV1.Service)
		client := c.clientset.CoreV1().Services(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing resource : %v ; error: config maps:%v", kind, err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			})
			if err != nil {
				log.Fatalf("resource update failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else {
			_, err := client.Create(req)
			if err != nil {
				log.Fatalf("resource creation failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		}
		c.waitForService(resource)
	}
}

func (c *GKE) serviceAccountApply(resource runtime.Object) {
	switch resource.GetObjectKind().GroupVersionKind().Version {
	case "v1":
		req := resource.(*apiCoreV1.ServiceAccount)
		client := c.clientset.CoreV1().ServiceAccounts(req.Namespace)
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing resource : %v ; error: config maps:%v", kind, err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			})
			if err != nil {
				log.Fatalf("resource update failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else {
			_, err := client.Create(req)

			if err != nil {
				log.Fatalf("resource creation failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		}
	}
}

func (c *GKE) clusterRoleApply(resource runtime.Object) {
	switch resource.GetObjectKind().GroupVersionKind().Version {
	case "v1":
		req := resource.(*rbac.ClusterRole)
		client := c.clientset.RbacV1().ClusterRoles()
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing resource : %v ; error: config maps:%v", kind, err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			})
			if err != nil {
				log.Fatalf("resource update failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else {
			_, err := client.Create(req)

			if err != nil {
				log.Fatalf("resource creation failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		}
	}
}

func (c *GKE) clusterRoleBindingApply(resource runtime.Object) {
	switch resource.GetObjectKind().GroupVersionKind().Version {
	case "v1":
		req := resource.(*rbac.ClusterRoleBinding)
		client := c.clientset.RbacV1().ClusterRoleBindings()
		kind := resource.GetObjectKind().GroupVersionKind().Kind

		list, err := client.List(apiMetaV1.ListOptions{})
		if err != nil {
			log.Fatalf("error listing resource : %v ; error: config maps:%v", kind, err)
		}

		var exists bool
		for _, l := range list.Items {
			if l.Name == req.Name {
				exists = true
				break
			}
		}

		if exists {
			err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				_, err := client.Update(req)
				return err
			})
			if err != nil {
				log.Fatalf("resource update failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource updated - kind: %v, name: %v", kind, req.Name)
		} else {
			_, err := client.Create(req)

			if err != nil {
				log.Fatalf("resource creation failed - kind: %v , error: %v", kind, err)
			}
			log.Printf("resource created - kind: %v, name: %v", kind, req.Name)
		}
	}
}
