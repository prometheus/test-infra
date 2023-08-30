# Prombench in EKS

Run prombench tests in [Elastic Kubernetes Service](https://aws.amazon.com/eks/).

## Setup prombench

1. [Create the main node](#create-the-main-node)
2. [Deploy monitoring components](#deploy-monitoring-components)

### Create the Main Node

---

- Create [security credentials](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html) on AWS. Put the credentials into a `yaml` file as follows:
```yaml
accesskeyid: <Amazon access key>
secretaccesskey: <Amazon access secret>
```
- Create a [VPC](https://docs.aws.amazon.com/eks/latest/userguide/create-public-private-vpc.html) with public subnets.
- Create a [Amazon EKS cluster role](https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html) with following policies:
    - AmazonEKSclusterPolicy 
- Create a [Amazon EKS worker node role](https://docs.aws.amazon.com/eks/latest/userguide/worker_node_IAM_role.html) with following policies:
    - AmazonEKSWorkerNodePolicy, AmazonEKS_CNI_Policy, AmazonEC2ContainerRegistryReadOnly
- Set the following environment variables and deploy the cluster.

```shell
export AUTH_FILE=<path to yaml credentials file that was created in the last step>
export CLUSTER_NAME=prombench
export ZONE=us-east-1
export EKS_WORKER_ROLE_ARN=<Amazon EKS worker node IAM role ARN>
export EKS_CLUSTER_ROLE_ARN=<Amazon EKS cluster role ARN>
# By default SEPARATOR DeploymentVar is set to `,` but you can override it by exporting and
# then passing it with the -v flag. It is used to split DeploymentVar into a slice.
export SEPARATOR=, 
export EKS_SUBNET_IDS=SUBNETID1,SUBNETID2,SUBNETID3
export PROVIDER=eks

make cluster_create
```


### Deploy monitoring components


> Collecting, monitoring and displaying the test results and logs
---

- [Optional] If used with the Github integration generate a GitHub auth token.
  - Login with the [Prombot account](https://github.com/prombot) and generate a [new auth token](https://github.com/settings/tokens).
  - With permissions: `public_repo`, `read:org`, `write:discussion`.

```shell
export GRAFANA_ADMIN_PASSWORD=password
export DOMAIN_NAME=prombench.prometheus.io // Can be set to any other custom domain or an empty string when not used with the Github integration.
export OAUTH_TOKEN=<generated token from github or set to an empty string " ">
export WH_SECRET=<github webhook secret>
export GITHUB_ORG=prometheus
export GITHUB_REPO=prometheus
```

- Deploy the [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx), Prometheus-Meta, Loki, Grafana, Alertmanager & Github Notifier.

```shell
make cluster_resource_apply
```

- The output will show the ingress IP which will be used to point the domain name to.
- Set the `A record` for `<DOMAIN_NAME>` to point to `nginx-ingress-controller` IP address.
- The services will be accessible at:
  - Grafana :: `http://<DOMAIN_NAME>/grafana`
  - Prometheus :: `http://<DOMAIN_NAME>/prometheus-meta`
  - Logs :: `http://<DOMAIN_NAME>/grafana/explore`
  - Profiles :: `http://<DOMAIN_NAME>/profiles`

## Usage

### Start a benchmarking test manually
---

- Set the following environment variables.

```shell
export RELEASE=<master/main or any prometheus release(ex: v2.3.0) >
export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
```

- Create the nodegroups for the k8s objects

```shell
make node_create
```

- Deploy the k8s objects

```shell
make resource_apply
```

### Stopping a benchmarking test manually

---

- Set the following environment variables:
```
export AUTH_FILE=<path to yaml credentials file that was created>
export CLUSTER_NAME=prombench
export SEPARATOR=,
export EKS_WORKER_ROLE_ARN=<Amazon EKS worker node IAM role ARN>
export EKS_CLUSTER_ROLE_ARN=<Amazon EKS cluster role ARN>
export EKS_SUBNET_IDS=SUBNETID1,SUBNETID2,SUBNETID3
export ZONE=us-east-1
export PROVIDER=eks

export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
```

- To delete just the nodepool (while keeping the cluster's main node intact), run:
```
make clean
```

- To delete everything (complete teardown of the entire cluster and all the resources), run:
```
make cluster_delete
```
