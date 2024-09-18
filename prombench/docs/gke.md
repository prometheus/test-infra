# Prombench in GKE

Run Prombench tests in [Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine).

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

1. **Create a New Project**: Start by creating a new project on Google Cloud.

2. **Create a Service Account**: 
    - Create a [Service Account](https://cloud.google.com/iam/docs/creating-managing-service-accounts) on GKE with the roles:
        - `Kubernetes Engine Service Agent`
        - `Kubernetes Engine Admin`
    - If using the `gcloud` CLI, add the following roles to the GCP Service Account:
        - [`roles/container.admin`](https://cloud.google.com/kubernetes-engine/docs/how-to/iam#kubernetes-engine-roles)
        - [`roles/iam.serviceAccountUser`](https://cloud.google.com/kubernetes-engine/docs/how-to/iam#service_account_user)
    - Download the JSON file for the service account.

3. **Set Environment Variables and Deploy the Cluster**:

    ```bash
    export GKE_PROJECT_ID=<google-cloud project-id>
    export CLUSTER_NAME=prombench
    export ZONE=us-east1-b
    export AUTH_FILE=<path to service-account.json>
    export PROVIDER=gke

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
    export SERVICEACCOUNT_CLIENT_EMAIL=<client-email present in service-account.json>
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
    export RELEASE=<master/main or any Prometheus release (e.g., v2.3.0)>
    export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
    ```

2. **Create Node Pools for Kubernetes Objects**:

    ```bash
    make node_create
    ```

3. **Deploy the Kubernetes Objects**:

    ```bash
    make resource_apply
    ```

### 2. Stopping a Benchmarking Test Manually

---

1. **Set the Environment Variables**:

    ```bash
    export GKE_PROJECT_ID=<google-cloud project-id>
    export CLUSTER_NAME=prombench
    export ZONE=us-east1-b
    export AUTH_FILE=<path to service-account.json>
    export PROVIDER=gke

    export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
    ```

2. **Delete Node Pools (Keeping the Main Node Intact)**:

    ```bash
    make clean
    ```

3. **Delete Everything (Complete Teardown)**:

    ```bash
    make cluster_delete
    ```
