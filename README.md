# prometheus-test-environment
A Kubernetes cluster preconfigured for testing Prometheus


## Pre-requisites
Make sure you have AWS credentials configured in a CLI profile. If you don't, then use `aws configure --profile <some-name>` to configure them.
For convenience, if these are the only credentials you use, you can leave out the `--profile` argument to get them configured as `default`.
If you do set a profile name, then make sure you `export AWS_PROFILE=<profile-name>` in the shell session where you want to use them.

## Building a cluster
Run `make` to create a cluster. This will create all the necesary resources in AWS using terraform and kops.
After the make command is finished, you cluster will take a little while to completely build and become available.
A `kubectl` context will be automatically configured for you with the credentials to access the cluster.
You can use it to check if the cluster is done building. Repeat `kubectl cluster-info` until you no longer get an error. Now your cluster is ready.

The automatic build process also deploys Prometheus to the cluster and preconfigures it for Kubernetes service discovery.

## Tearing down a cluster.
Run `make clean` to destroy the cluster and any associated resources on AWS.
Once you're done using the cluster, this will make sure that no resources are left running that can generate unnecesary costs.
If you get any kind of error during the `clean` run, please double check which resources may have failed to be destroyed on AWS manually.

## Cluster settings
The K8s cluster is created using Kops. This is a fairly opinionated tool and not every aspect of the AWS infrastructure or Kubernetes can be customized.
For tweaking the attributes of the nodes, please have a look at the YAML files in `manifests/kops`. These are Kops manifests describing different instance groups.
You can use them to tweak instance types, number of nodes of each type, labels applied to each node, etc.

Kubernetes manifests used to deploy the default Prometheus and it's configuration are stored under `manifests/k8s`.
Here you can tweak the Prometheus deployment and the configuration file passed to it.