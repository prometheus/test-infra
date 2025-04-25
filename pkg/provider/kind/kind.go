// Copyright 2020 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kind

import (
	"context"
	"fmt"
	"strings"

	"gopkg.in/alecthomas/kingpin.v2"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cmd"

	"github.com/prometheus/test-infra/pkg/provider"
	k8sProvider "github.com/prometheus/test-infra/pkg/provider/k8s"
)

type Resource = provider.Resource

// KIND holds the fields used to generate an API request.
type KIND struct {
	// The k8s provider used when we work with the manifest files.
	k8sProvider *k8sProvider.K8s
	// The kind provider used to instantiate a new provider.
	kindProvider *cluster.Provider
	// Final DeploymentFiles files.
	DeploymentFiles []string
	// Final DeploymentVars.
	DeploymentVars map[string]string
	// DeployResource to construct DeploymentVars and DeploymentFiles
	DeploymentResource *provider.DeploymentResource
	// Content bytes after parsing the template variables, grouped by filename.
	kindResources []Resource
	// K8s resource.runtime objects after parsing the template variables, grouped by filename.
	k8sResources []k8sProvider.Resource

	ctx context.Context
	// KIND kuberconfig file
	kubeconfig string
}

// New is the KIND constructor.
func New(dr *provider.DeploymentResource) *KIND {
	return &KIND{
		DeploymentResource: dr,
		kindProvider: cluster.NewProvider(
			cluster.ProviderWithLogger(cmd.NewLogger()),
		),
		ctx:        context.Background(),
		kubeconfig: homedir.HomeDir() + "/.kube/config",
	}
}

// SetupDeploymentResources Sets up DeploymentVars and DeploymentFiles
func (c *KIND) SetupDeploymentResources(*kingpin.ParseContext) error {
	customDeploymentVars := map[string]string{
		"NGINX_SERVICE_TYPE":        "NodePort",
		"LOADGEN_SCALE_UP_REPLICAS": "2",
	}

	c.DeploymentFiles = c.DeploymentResource.DeploymentFiles
	c.DeploymentVars = provider.MergeDeploymentVars(
		c.DeploymentResource.DefaultDeploymentVars,
		customDeploymentVars,
		c.DeploymentResource.FlagDeploymentVars,
	)
	return nil
}

// The CreateNamespace function is used to create the PR namespace and copy the
// blocksync-config and bucket-secret from the default namespace to the prombench-${PR_NUMBER} namespace.
// Block-sync uses these resources to download data from object storage.
// For more information, refer to this PR: https://github.com/prometheus/test-infra/pull/840

func (c *KIND) CreateNamespace(*kingpin.ParseContext) error {
	sourceNS := "default"
	targetNS := "prombench-" + c.DeploymentVars["PR_NUMBER"]
	configMapName := "blocksync-config"
	secretName := "bucket-secret"

	// check if namespace exists
	_, err := c.k8sProvider.Clt.CoreV1().Namespaces().Get(context.TODO(), targetNS, metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			ns := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: targetNS,
				},
			}
			_, err = c.k8sProvider.Clt.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("error creating namespace: %w", err)
			}
		} else {
			return fmt.Errorf("error checking namespace: %w", err)
		}
	}

	// copy ConfigMap
	_, err = c.k8sProvider.Clt.CoreV1().ConfigMaps(targetNS).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		cm, err := c.k8sProvider.Clt.CoreV1().ConfigMaps(sourceNS).Get(context.TODO(), configMapName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting configmap: %w", err)
		}
		cm.ResourceVersion = ""
		cm.Namespace = targetNS
		_, err = c.k8sProvider.Clt.CoreV1().ConfigMaps(targetNS).Create(context.TODO(), cm, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating configmap: %w", err)
		}
	}

	// copy Secret
	_, err = c.k8sProvider.Clt.CoreV1().Secrets(targetNS).Get(context.TODO(), secretName, metav1.GetOptions{})
	if k8serrors.IsNotFound(err) {
		secret, err := c.k8sProvider.Clt.CoreV1().Secrets(sourceNS).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error getting secret: %w", err)
		}
		secret.ResourceVersion = ""
		secret.Namespace = targetNS
		_, err = c.k8sProvider.Clt.CoreV1().Secrets(targetNS).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("error creating secret in target NS: %w", err)
		}
	}

	return nil
}

// KINDDeploymentsParse parses the environment/kind deployment files and saves the result as bytes grouped by the filename.
// Any DeploymentVar will be replaced in the resources files following the golang text template format.
func (c *KIND) KINDDeploymentsParse(*kingpin.ParseContext) error {
	if err := c.checkDeploymentVarsAndFiles(); err != nil {
		return err
	}

	deploymentResource, err := provider.DeploymentsParse(c.DeploymentFiles, c.DeploymentVars)
	if err != nil {
		return err
	}
	c.kindResources = deploymentResource
	return nil
}

func (c *KIND) K8SDeploymentsParse(*kingpin.ParseContext) error {
	if err := c.checkDeploymentVarsAndFiles(); err != nil {
		return err
	}

	deploymentResource, err := provider.DeploymentsParse(c.DeploymentFiles, c.DeploymentVars)
	if err != nil {
		return err
	}
	for _, deployment := range deploymentResource {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		k8sObjects := make([]runtime.Object, 0)

		for _, text := range strings.Split(string(deployment.Content), provider.Separator) {
			text = strings.TrimSpace(text)
			if len(text) == 0 {
				continue
			}

			resource, _, err := decode([]byte(text), nil, nil)
			if err != nil {
				return fmt.Errorf("decoding the resource file:%v, section:%v...: %w", deployment.FileName, text[:100], err)
			}
			if resource == nil {
				continue
			}
			k8sObjects = append(k8sObjects, resource)
		}
		if len(k8sObjects) > 0 {
			c.k8sResources = append(c.k8sResources, k8sProvider.Resource{FileName: deployment.FileName, Objects: k8sObjects})
		}
	}
	return nil
}

// checkDeploymentVarsAndFiles checks whether the requied deployment vars are passed.
func (c *KIND) checkDeploymentVarsAndFiles() error {
	reqDepVars := []string{"CLUSTER_NAME"}
	for _, k := range reqDepVars {
		if v, ok := c.DeploymentVars[k]; !ok || v == "" {
			return fmt.Errorf("missing required %v variable", k)
		}
	}
	if len(c.DeploymentFiles) == 0 {
		return fmt.Errorf("missing deployment file(s)")
	}
	return nil
}

// ClusterCreate create a new cluster or applies changes to an existing cluster.
func (c *KIND) ClusterCreate(*kingpin.ParseContext) error {
	for _, deployment := range c.kindResources {
		CreateWithConfigFile := cluster.CreateWithRawConfig(deployment.Content)

		err := c.kindProvider.Create(c.DeploymentVars["CLUSTER_NAME"], CreateWithConfigFile)
		if err != nil {
			return err
		}
	}
	return nil
}

// ClusterDelete deletes a k8s cluster.
func (c *KIND) ClusterDelete(*kingpin.ParseContext) error {
	err := c.kindProvider.Delete(c.DeploymentVars["CLUSTER_NAME"], c.kubeconfig)
	if err != nil {
		return err
	}
	return nil
}

// NewK8sProvider sets the k8s provider used for deploying k8s manifests.
func (c *KIND) NewK8sProvider(*kingpin.ParseContext) error {
	var err error
	apiConfig, err := clientcmd.LoadFromFile(c.kubeconfig)
	if err != nil {
		return err
	}

	c.k8sProvider, err = k8sProvider.New(c.ctx, apiConfig)
	if err != nil {
		return err
	}
	return nil
}

// ResourceApply calls k8s.ResourceApply to apply the k8s objects in the manifest files.
func (c *KIND) ResourceApply(*kingpin.ParseContext) error {
	if err := c.k8sProvider.ResourceApply(c.k8sResources); err != nil {
		return err
	}
	return nil
}

// ResourceDelete calls k8s.ResourceDelete to apply the k8s objects in the manifest files.
func (c *KIND) ResourceDelete(*kingpin.ParseContext) error {
	if err := c.k8sProvider.ResourceDelete(c.k8sResources); err != nil {
		return err
	}
	return nil
}

// GetDeploymentVars shows deployment variables.
func (c *KIND) GetDeploymentVars(_ *kingpin.ParseContext) error {
	fmt.Print("-------------------\n   DeploymentVars   \n------------------- \n")
	for key, value := range c.DeploymentVars {
		fmt.Println(key, ": ", value)
	}
	return nil
}
