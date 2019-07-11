package provider

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const (
	GlobalRetryCount = 30
	Separator        = "---"
	globalRetryTime  = 10 * time.Second
)

// Resource holds the file content after parsing the template variables.
type Resource struct {
	FileName string
	Content  []byte
}

// RetryUntilTrue returns when there is an error or the requested operation returns true.
func RetryUntilTrue(name string, retryCount int, fn func() (bool, error)) error {
	for i := 1; i <= retryCount; i++ {
		if ready, err := fn(); err != nil {
			return err
		} else if !ready {
			log.Printf("Request for '%v' is in progress. Checking in %v", name, globalRetryTime)
			time.Sleep(globalRetryTime)
			continue
		}
		log.Printf("Request for '%v' is done!", name)
		return nil
	}
	return fmt.Errorf("Request for '%v' hasn't completed after retrying %d times", name, retryCount)
}

// applyTemplateVars applies golang templates to deployment files.
func applyTemplateVars(file string, deploymentVars map[string]string) ([]byte, error) {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatalf("Error reading file %v:%v", file, err)
	}

	fileContentParsed := bytes.NewBufferString("")
	t := template.New("resource").Option("missingkey=error")
	// k8s objects can't have dots(.) se we add a custom function to allow normalising the variable values.
	t = t.Funcs(template.FuncMap{
		"normalise": func(t string) string {
			return strings.Replace(t, ".", "-", -1)
		},
	})
	if err := template.Must(t.Parse(string(content))).Execute(fileContentParsed, deploymentVars); err != nil {
		log.Fatalf("Failed to execute parse file:%s err:%v", file, err)
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
	for k, v := range deploymentVars {
		_, err := os.Stat(v)
		fmt.Printf("%v %v", k, v)
		if err == nil && v != "prombench" {
			log.Printf("file exists")
			val, e := ioutil.ReadFile(v)
			fmt.Printf(string(val))
			if e != nil {
				log.Fatalf("couldn't read var file")
			}
			deploymentVars[k] = string(val)
		} else {
			log.Printf("file doesn't exist and command line variable and not stored in file")
		}
	}
	fmt.Printf("%v+", deploymentVars)
	deploymentObjects := make([]Resource, 0)
	for _, name := range fileList {
		content, err := applyTemplateVars(name, deploymentVars)
		if err != nil {
			return nil, fmt.Errorf("couldn't apply template to file %s: %v", name, err)
		}
		deploymentObjects = append(deploymentObjects, Resource{FileName: name, Content: content})
	}
	return deploymentObjects, nil
}
