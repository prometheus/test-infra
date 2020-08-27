// Copyright 2019 The Prometheus Authors
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

package provider

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const (
	GlobalRetryCount = 50
	Separator        = "---"
	globalRetryTime  = 10 * time.Second
)

// DeploymentResource holds list of variables and corresponding files.
type DeploymentResource struct {
	// DeploymentFiles files provided from the cli.
	DeploymentFiles []string
	// DeploymentVars provided from the cli.
	FlagDeploymentVars map[string]string
	// Default DeploymentVars.
	DefaultDeploymentVars map[string]string
	// Dashboards
	Dashboards map[string]string
}

// NewDeploymentResource returns DeploymentResource with default values.
func NewDeploymentResource() *DeploymentResource {
	return &DeploymentResource{
		DeploymentFiles:    []string{},
		FlagDeploymentVars: map[string]string{},
		DefaultDeploymentVars: map[string]string{
			"NGINX_SERVICE_TYPE": "LoadBalancer",
		},
	}
}

// Resource holds the file content after parsing the template variables.
type Resource struct {
	FileName string
	Content  []byte
}

// RetryUntilTrue returns when there is an error or the requested operation returns true.
func RetryUntilTrue(name string, retryCount int, fn func() (bool, error)) error {
	for i := 1; i <= retryCount; i++ {
		time.Sleep(globalRetryTime)
		if ready, err := fn(); err != nil {
			return err
		} else if !ready {
			log.Printf("Request for '%v' is in progress. Checking in %v", name, globalRetryTime)
			continue
		}
		log.Printf("Request for '%v' is done!", name)
		return nil
	}
	return fmt.Errorf("Request for '%v' hasn't completed after retrying %d times", name, retryCount)
}

// applyTemplateVars applies golang templates to deployment files.
func applyTemplateVars(content []byte, deploymentVars map[string]string) ([]byte, error) {
	fileContentParsed := bytes.NewBufferString("")
	t := template.New("resource").Option("missingkey=error")
	// k8s objects can't have dots(.) se we add a custom function to allow normalising the variable values.
	t = t.Funcs(template.FuncMap{
		"normalise": func(t string) string {
			return strings.Replace(t, ".", "-", -1)
		},
	})
	if err := template.Must(t.Parse(string(content))).Execute(fileContentParsed, deploymentVars); err != nil {
		return nil, fmt.Errorf("Failed to execute parse file err: %s", err)
	}
	return fileContentParsed.Bytes(), nil
}

// DeploymentsParse parses the deployment files and returns the result as bytes grouped by the filename.
// Any variables passed to the cli will be replaced in the resources files following the golang text template format.
func DeploymentsParse(deploymentFiles []string, deploymentVars map[string]string) ([]Resource, error) {
	var fileList []string
	for _, name := range deploymentFiles {
		if file, err := os.Stat(name); err == nil && file.IsDir() {
			if err := filepath.Walk(name, func(path string, f os.FileInfo, err error) error {
				if filepath.Ext(path) == ".yaml" || filepath.Ext(path) == ".yml" {
					fileList = append(fileList, path)
				}
				return nil
			}); err != nil {
				return nil, fmt.Errorf("error reading directory: %v", err)
			}
		} else {
			fileList = append(fileList, name)
		}
	}

	deploymentObjects := make([]Resource, 0)
	for _, name := range fileList {
		absFileName := strings.TrimSuffix(filepath.Base(name), filepath.Ext(name))
		content, err := ioutil.ReadFile(name)
		if err != nil {
			log.Fatalf("Error reading file %v:%v", name, err)
		}
		// Don't parse file with the suffix "noparse".
		if !strings.HasSuffix(absFileName, "noparse") {
			content, err = applyTemplateVars(content, deploymentVars)
			if err != nil {
				return nil, fmt.Errorf("couldn't apply template to file %s: %v", name, err)
			}
		}
		deploymentObjects = append(deploymentObjects, Resource{FileName: name, Content: content})
	}
	return deploymentObjects, nil
}

// MergeDeploymentVars merges multiple maps based on the order.
func MergeDeploymentVars(ms ...map[string]string) map[string]string {
	res := map[string]string{}
	for _, m := range ms {
		for k, v := range m {
			res[k] = v
		}
	}
	return res
}

func getK8sClientSet() (*kubernetes.Clientset, error) {
	kubeconfig := flag.String("kubeconfig", "/home/raj/.kube/config", "absolute path to the kubeconfig file")

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func CreateGrafanaDashboardsConfigMap() error {

	clientset, err := getK8sClientSet()
	if err != nil {
		return err
	}

	NodeMetrics, err := ioutil.ReadFile("manifests/dashboards/node-metrics.json")
	if err != nil {
		return nil
	}
	PromBench, err := ioutil.ReadFile("manifests/dashboards/prombench.json")
	if err != nil {
		return nil
	}

	ConfigMapData := map[string]string{
		"node-metrics.json": string(NodeMetrics),
		"prombench.json":    string(PromBench),
	}

	newConfigMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "grafana-dashboards",
		},
		Data: ConfigMapData,
	}

	_, err = clientset.CoreV1().ConfigMaps("default").Create(context.TODO(), &newConfigMap, metav1.CreateOptions{})
	if err != nil {
		return nil
	}

	log.Printf("resource created - kind: ConfigMap, name: grafana-dashboards")
	return nil
}

func DeleteGrafanaDashboardsConfigMap() error {

	clientset, err := getK8sClientSet()
	if err != nil {
		return err
	}

	err = clientset.CoreV1().ConfigMaps("default").Delete(context.TODO(), "grafana-dashboards", metav1.DeleteOptions{})
	if err != nil {
		return err
	}

	log.Printf("resource created - kind: ConfigMap, name: grafana-dashboards")
	return nil
}
