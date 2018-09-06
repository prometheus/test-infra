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
	globalRetryTime  = 10 * time.Second
)

// ResourceFile is a common struct to save name and content of input files
type ResourceFile struct {
	Name    string
	Content []byte
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

// ApplyTemplateVars applies golang templates to deployment files
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
func DeploymentsParse(deploymentFiles []string, deploymentVars map[string]string) ([]ResourceFile, error) {
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

	deploymentsContent := make([]ResourceFile, 0)
	for _, name := range fileList {
		content, err := applyTemplateVars(name, deploymentVars)
		if err != nil {
			return nil, fmt.Errorf("couldn't apply template to file %s: %v", name, err)
		}
		deploymentsContent = append(deploymentsContent, ResourceFile{Name: name, Content: content})
	}
	return deploymentsContent, nil
}
