# Prombench in EKS

Run Prombench tests in [Elastic Kubernetes Service (EKS)](https://aws.amazon.com/eks/).

## Table of Contents

1. [Setup Prombench](#setup-prombench)
    - [Create the Main Node](#create-the-main-node)
    - [Deploy Monitoring Components](#deploy-monitoring-components)
2. [Usage](#usage)
    - [Start a Benchmarking Test Manually](#start-a-benchmarking-test-manually)
    - [Stopping a Benchmarking Test Manually](#stopping-a-benchmarking-test-manually)

## Setup Prombench

### 1. Create the Main Node

---

1. **Create Security Credentials**:
    - Create [security credentials](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html) on AWS.
    - Store the credentials in a YAML file as follows:

    ```yaml
    accesskeyid: <Amazon access key>
    secretaccesskey: <Amazon access secret>
    ```

2. **Create a VPC**:
    - Set up a [VPC](https://docs.aws.amazon.com/eks/latest/userguide/create-public-private-vpc.html) with public subnets.

3. **Create IAM Roles**:
    - **EKS Cluster Role**: Create an [Amazon EKS cluster role](https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html) with the following policy:
        - `AmazonEKSclusterPolicy`
    - **EKS Worker Node Role**: Create an [Amazon EKS worker node role](https://docs.aws.amazon.com/eks/latest/userguide/worker_node_IAM_role.html) with the following policies:
        - `AmazonEKSWorkerNodePolicy`
        - `AmazonEKS_CNI_Policy`
        - `AmazonEC2ContainerRegistryReadOnly`

4. **Set Environment Variables and Deploy the Cluster**:

    ```bash
    export AUTH_FILE=<path to yaml credentials file that was created in the last step>
    export CLUSTER_NAME=prombench
    export ZONE=us-east-1
    export EKS_WORKER_ROLE_ARN=<Amazon EKS worker node IAM role ARN>
    export EKS_CLUSTER_ROLE_ARN=<Amazon EKS cluster role ARN>
    export SEPARATOR=, 
    export EKS_SUBNET_IDS=SUBNETID1,SUBNETID2,SUBNETID3
    export PROVIDER=eks

    make cluster_create
    ```

### 2. Deploy Monitoring Components

---

> **Note**: These components are responsible for collecting, monitoring, and displaying test results and logs.

1. **Optional GitHub Integration**:
    - If used with GitHub integration, generate a GitHub auth token:
        - Login with the [Prombot account](https://github.com/prombot) and generate a [new auth token](https://github.com/settings/tokens).
        - Required permissions: `public_repo`, `read:org`, `write:discussion`.

    ```bash
    export GRAFANA_ADMIN_PASSWORD=password
    export DOMAIN_NAME=prombench.prometheus.io # Can be set to any other custom domain or an empty string if not used with the GitHub integration.
    export OAUTH_TOKEN=<generated token from GitHub or set to an empty string " ">
    export WH_SECRET=<GitHub webhook secret>
    export GITHUB_ORG=prometheus
    export GITHUB_REPO=prometheus
    ```

2. **Deploy the Monitoring Components**:
    - This step will deploy the [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx), Prometheus-Meta, Loki, Grafana, Alertmanager, and GitHub Notifier.

    ```bash
    make cluster_resource_apply
    ```

3. **Configure DNS**:
    - The output will display the ingress IP. Use this IP to point the domain name.
    - Set the `A record` for `<DOMAIN_NAME>` to point to the `nginx-ingress-controller` IP address.

4. **Access the Services**:
    - Grafana: `http://<DOMAIN_NAME>/grafana`
    - Prometheus: `http://<DOMAIN_NAME>/prometheus-meta`
    - Logs: `http://<DOMAIN_NAME>/grafana/explore`
    - Profiles: `http://<DOMAIN_NAME>/profiles`

## Usage

### 1. Start a Benchmarking Test Manually

---

1. **Set the Environment Variables**:

    ```bash
    export RELEASE=<master/main or any prometheus release (e.g., v2.3.0)>
    export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
    ```

2. **Create Nodegroups for Kubernetes Objects**:

    ```bash
    make node_create
    ```
3. **Setting Up Benchmarking Data**    

 When setting up a benchmarking environment, it’s often useful to have pre-generated data available. This data can help speed up testing and make benchmarks more realistic by simulating actual workloads.

In this setup, you have two choices:

Here’s how each option works:
- **Option 1: Download data from object storage**

   To download data from object storage, create a Kubernetes secret with exact named `bucket-secret` and file name `object-config.yml`  with the necessary credentials as per your object storage. This secret enables access to the stored data.
> Note: Make sure this secret applied before ```3_prometheus-test-pr_deployment.yaml``` and ```3_prometheus-test-release_deployment.yaml```

- **Option 2: Skip downloading data**

If you don’t Want to Download data create an empty secret like this -

```yaml
# Empty Secret to Skip Downloading Data
apiVersion: v1
kind: Secret
metadata:
  name: bucket-secret
  namespace: prombench-{{ .PR_NUMBER }} 
type: Opaque
stringData:
  object-config.yml: 
```  
 
Regardless of the option chosen, data stored in Prometheus will only be retained based on the configured retention settings (```--storage.tsdb.retention.size```). 

> **⚠️ Warning:** The benchmark will change its basis when the retention size limit is reached and older downloaded blocks are deleted. Ensure that you have sufficient retention settings configured to avoid data loss that could affect benchmarking results. 

4. **Downloading Directory configuration**

PromBench can download data from a specific directory in object storage based on a configuration file. This configuration file specifies the directory name along with the minimum and maximum timestamps.
> **Note:** Make sure the file is applied before the ```3_prometheus-test-pr_deployment.yaml``` and ```3_prometheus-test-release_deployment.yaml``` . 

 - **Option 1: To Download Data from a Specific Directory**

 Create a ConfigMap with the following structure:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: blocksync-config
  namespace: prombench-{{ .PR_NUMBER }}
data:
  bucket-config.yml: |
    path: your-directory-name
    minTime: block-starting-time
    maxTime: block-ending-time
```
Replace the values of ```directory```,```minTime```, and ```maxTime``` with your desired configuration. Here ```minTime``` , ```maxTime``` are the starting and ending time of TSDB block and in Unix timestamp format. 
- **Option 2: To Skip Data Download**

If you do not want to download data, create an empty ConfigMap with the same name:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: blocksync-config
  namespace: prombench-{{ .PR_NUMBER }}
data:
```

5. **Deploy the Kubernetes Objects**:

    ```bash
    make resource_apply
    ```

### 2. Stopping a Benchmarking Test Manually

---

1. **Set the Environment Variables**:

    ```bash
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

2. **Delete Nodegroups (Keeping the Main Node Intact)**:

    ```bash
    make clean
    ```

3. **Delete Everything (Complete Teardown)**:

    ```bash
    make cluster_delete
    ```
