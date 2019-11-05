# Automated Prometheus E2E testing and benchmarking.

![Prombench Design](design.svg)

It runs with [Github Actions](https://github.com/features/actions) on a [Google Kubernetes Engine Cluster](https://cloud.google.com/kubernetes-engine/).
It is designed to support adding more k8s providers.

## Overview of the manifest files
The `/manifest` directory contains all the kubernetes manifest files.
- `cluster.yaml` : This is used to create the Main Node.
- `cluster-infra/` : These are the persistent components of the Main Node.
- `prombench/` : These resources are created and destroyed for each prombench test.

## Setup prombench
1. [Create the main node](#create-the-main-node)
2. [Deploy monitoring components](#deploy-monitoring-components)
3. [Setup GitHub Actions](#setup-github-actions)

### Create the Main Node
---
- Create a new project on Google Cloud.
- Create a [Service Account](https://cloud.google.com/iam/docs/creating-managing-service-accounts) on GKE with role `Kubernetes Engine Service Agent` & `Kubernetes Engine Admin`. If using gcloud cli add the [`roles/container.admin`](https://cloud.google.com/kubernetes-engine/docs/how-to/iam#kubernetes-engine-roles) and [`roles/iam.serviceAccountUser`](https://cloud.google.com/kubernetes-engine/docs/how-to/iam#service_account_user) roles to the GCP serviceAccount and download the json file.

- Set the following environment variables and deploy the cluster.
```
export PROJECT_ID=<google-cloud project-id>
export CLUSTER_NAME=prombench
export ZONE=us-east1-b
export AUTH_FILE=<path to service-account.json>

./prombench gke cluster create -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID \
    -v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME -f manifests/cluster.yaml
```

### Deploy monitoring components
> Collecting, monitoring and displaying the test results and logs

---

- [Optional] If used with the Github integration generate a GitHub auth token.
  * Login with the [Prombot account](https://github.com/prombot) and generate a [new auth token](https://github.com/settings/tokens).
  * With permissions: `public_repo`, `read:org`, `write:discussion`.
```
export GCLOUD_SERVICEACCOUNT_CLIENT_EMAIL=<client-email present in service-account.json>
export GRAFANA_ADMIN_PASSWORD=password
export DOMAIN_NAME=prombench.prometheus.io // Can be set to any other custom domain or an empty string when not used with the Github integration.
export OAUTH_TOKEN=<generated token from github or set to an empty string " ">
export GITHUB_ORG=prometheus
export GITHUB_REPO=prometheus
```

- Deploy the [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx), Prometheus-Meta, Loki, Grafana, Alertmanager & Github Notifier.
```
./prombench gke resource apply -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID -v ZONE:$ZONE \
    -v CLUSTER_NAME:$CLUSTER_NAME -v DOMAIN_NAME:$DOMAIN_NAME \
    -v GRAFANA_ADMIN_PASSWORD:$GRAFANA_ADMIN_PASSWORD \
    -v GCLOUD_SERVICEACCOUNT_CLIENT_EMAIL:$GCLOUD_SERVICEACCOUNT_CLIENT_EMAIL \
    -v OAUTH_TOKEN="$(printf $OAUTH_TOKEN | base64 -w 0)" \
    -v GKE_AUTH="$(cat $AUTH_FILE | base64 -w 0)" \
    -v GITHUB_ORG:$GITHUB_ORG -v GITHUB_REPO:$GITHUB_REPO \
    -f manifests/cluster-infra
```
> Note: Use `-v GKE_AUTH="$(echo $AUTH_FILE | base64 -w 0)"` if you're passing the data directly into `$AUTH_FILE`
- The output will show the ingress IP which will be used to point the domain name to. Alternatively you can see it from the GKE/Services tab.
- Set the `A record` for `<DOMAIN_NAME>` to point to `nginx-ingress-controller` IP address.
- The services will be accessible at:
  * Grafana :: `http://<DOMAIN_NAME>/grafana`
  * Prometheus ::  `http://<DOMAIN_NAME>/prometheus-meta`
  * Logs :: `http://<DOMAIN_NAME>/grafana/explore`

### Setup GitHub Actions
Place a workflow file in the `.github` directory of the repository.
See the [prometheus/prometheus](https://github.com/prometheus/prometheus) repository for an example.

Create a github action `PROMBENCH_GKE_AUTH` secret with the base64 encoded content of the `service-account.json` file.
```
cat $AUTH_FILE | base64 -w 0
```

## Usage
### Trigger tests via a Github comment.
---
> Due to the high cost of each test, only maintainers can manage tests.

**Starting:**
- `/prombench master` - compare PR with the master branch.
- `/prombench v2.4.0` - compare PR with a release version, from [quay.io/prometheus/prometheus:releaseVersion](https://quay.io/prometheus/prometheus:releaseVersion)

**Restarting:**
- `/prombench restart <release_version>`

**Stopping:**
- `/prombench cancel`

### Start a benchmarking test manually
---

- Set the following environment variables.
```
export RELEASE=<master or any prometheus release(ex: v2.3.0) >
export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
```

- Create the nodepools for the k8s objects
```
./prombench gke nodepool create -a $AUTH_FILE \
    -v ZONE:$ZONE -v PROJECT_ID:$PROJECT_ID -v CLUSTER_NAME:$CLUSTER_NAME \
    -v PR_NUMBER:$PR_NUMBER -f manifests/prombench/nodepools.yaml
```

- Deploy the k8s objects
```
./prombench gke resource apply -a $AUTH_FILE \
    -v ZONE:$ZONE -v PROJECT_ID:$PROJECT_ID -v CLUSTER_NAME:$CLUSTER_NAME \
    -v PR_NUMBER:$PR_NUMBER -v RELEASE:$RELEASE -v DOMAIN_NAME:$DOMAIN_NAME \
    -v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
    -f manifests/prombench/benchmark
```
