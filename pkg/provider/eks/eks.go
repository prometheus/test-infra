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

package eks

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	awsSession "github.com/aws/aws-sdk-go/aws/session"
	eks "github.com/aws/aws-sdk-go/service/eks"
	"github.com/pkg/errors"
	k8sProvider "github.com/prometheus/test-infra/pkg/provider/k8s"

	"github.com/prometheus/test-infra/pkg/provider"
	"gopkg.in/alecthomas/kingpin.v2"
)

type Resource = provider.Resource

// EKS holds the fields used to generate an API request.
type EKS struct {
	ClusterName string
	// The eks client used when performing EKS requests.
	clientEKS *eks.EKS
	// The k8s provider used when we work with the manifest files.
	k8sProvider *k8sProvider.K8s
	// DeploymentFiles files provided from the cli.
	DeploymentFiles []string
	// Variables to substitute in the DeploymentFiles.
	// These are also used when the command requires some variables that are not provided by the deployment file.
	DeploymentVars map[string]string
	// Content bytes after parsing the template variables, grouped by filename.
	eksResources []Resource
	// K8s resource.runtime objects after parsing the template variables, grouped by filename.
	k8sResources []k8sProvider.Resource
}

// New is the EKS constructor
func New() *EKS {
	return &EKS{
		DeploymentVars: make(map[string]string),
	}
}

// NewEKSClient sets the EKS client used when performing the GKE requests.
func (c *EKS) NewEKSClient(*kingpin.ParseContext) error {
	if os.Getenv("AWS_ACCESS_KEY_ID") == "" || os.Getenv("AWS_SECRET_ACCESS_KEY") == "" {
		return errors.Errorf("no auth provided! Need to set the AWS_ACCCESS_KEY_ID and AWS_SECRET_ACCESS_KEY env variable")
	}

	cl := eks.New(awsSession.Must(awsSession.NewSession()), aws.NewConfig())
	c.clientEKS = cl
	return nil
}
