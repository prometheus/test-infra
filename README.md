# Automated Prometheus E2E testing and benchmarking.

![Prombench Design](design.svg)

It runs with [Prow CI](https://github.com/kubernetes/test-infra/blob/master/prow/) on a [Google Kubernetes Engine Cluster](https://cloud.google.com/kubernetes-engine/).
It is designed to support adding more k8s providers.

## Overview of the manifest files
The `/manifest` directory contains all the kubernetes manifest files.
- `cluster.yaml` : This is used to create the GKE cluster.
- `cluster-infra/` : These are the persistent cluster infrastructure resources.
- `prombench/` : These resources are created and destoryed for each prombench test.
- `prow/` : Resources for deploying [prow](https://github.com/kubernetes/test-infra/tree/master/prow/), which is used to trigger tests from GitHub comments.

## Setting up the test-infra
### Create a k8s cluster
---
- Create a new project on Google Cloud.
- Create a [Service Account](https://cloud.google.com/iam/docs/creating-managing-service-accounts) on GKE with role `Kubernetes Engine Service Agent` & `Kubernetes Engine Admin`. If using gcloud cli add the [`roles/container.admin`](https://cloud.google.com/kubernetes-engine/docs/how-to/iam#kubernetes-engine-roles) and [`roles/iam.serviceAccountUser`](https://cloud.google.com/kubernetes-engine/docs/how-to/iam#service_account_user) roles to the GCP serviceAccount and download the json file.

- Set the following environment variables & deploy cluster.
```
export PROJECT_ID=<google-cloud project-id>
export CLUSTER_NAME=prombench
export ZONE=us-east1-b
export AUTH_FILE=<path to service-account.json>
export GOOGLE_APPLICATION_CREDENTIALS=$AUTH_FILE

./prombench gke cluster create -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID \
    -v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME -f manifests/cluster.yaml
```

> `GOOGLE_APPLICATION_CREDENTIALS` is needed for the k8s provider after [#222](https://github.com/prometheus/prombench/pull/222), long term plan is to remove this env var and pass the value of the file directly to the k8s provider.

### Deploy monitoring infra
> This is used for collecting, monitoring and displaying the test results

---

- Generate a GitHub auth token that will be used to authenticate when sending requests to the GitHub API.
  * Login with the [Prombot account](https://github.com/prombot) and generate a [new auth token](https://github.com/settings/tokens).
  * With permissions: `public_repo`, `read:org`, `write:discussion`.
- Set the following environment variables
```
export GCLOUD_SERVICEACCOUNT_CLIENT_EMAIL=<client-email present in service-account.json>
export GRAFANA_ADMIN_PASSWORD=password
export DOMAIN_NAME=prombench.prometheus.io // Can be set to any other custom domain.
export OAUTH_TOKEN=<generated token from github>
```

- Deploy the [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx), Prometheus-Meta, Grafana, Alertmanager & amGithubNotifier.
```
./prombench gke resource apply -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID -v ZONE:$ZONE \
    -v CLUSTER_NAME:$CLUSTER_NAME -v DOMAIN_NAME:$DOMAIN_NAME \
    -v GRAFANA_ADMIN_PASSWORD:$GRAFANA_ADMIN_PASSWORD \
    -v GCLOUD_SERVICEACCOUNT_CLIENT_EMAIL:$GCLOUD_SERVICEACCOUNT_CLIENT_EMAIL \
    -v OAUTH_TOKEN="$(printf $OAUTH_TOKEN | base64 -w 0)" \
    -f manifests/cluster-infra
```
- The output will show the ingress IP which will be used to point the domain name to. Alternatively you can see it from the GKE/Services tab.
- Set the `A record` for `<DOMAIN_NAME>` to point to `nginx-ingress-controller` IP address.
- The services will be accessible at:
  * Grafana :: `http://<DOMAIN_NAME>/grafana`
  * Prometheus ::  `http://<DOMAIN_NAME>/prometheus-meta`

### Deploy Prow
> This is used to monitor GitHub comments and starts new tests.

---
- Set the following environment variables
```
export HMAC_TOKEN=$(openssl rand -hex 20)
```

- Add a [github webhook](https://github.com/prometheus/prometheus/settings/hooks) where to send the events.
  * Content Type: `json`
  * Send:  `Issue comments,Pull requests`
  * Secret: `echo $HMAC_TOKEN`
  * Payload URL: `http://<DOMAIN_NAME>/hook`

- Add all required tokens as k8s secrets.
  * hmac is used when verifying requests from GitHub.
  * gke auth is used when scaling up and down the cluster.
```
./prombench gke resource apply -a $AUTH_FILE -v ZONE:$ZONE \
    -v CLUSTER_NAME:$CLUSTER_NAME -v PROJECT_ID:$PROJECT_ID \
    -v HMAC_TOKEN="$(printf $HMAC_TOKEN | base64 -w 0)" \
    -v GKE_AUTH="$(cat $AUTH_FILE | base64 -w 0)" \
    -f manifests/prow/secrets.yaml
```

- Deploy all internal prow components

```
export GITHUB_ORG=prometheus
export GITHUB_REPO=prometheus

./prombench gke resource apply -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID \
    -v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME -v DOMAIN_NAME:$DOMAIN_NAME \
    -v GITHUB_ORG:$GITHUB_ORG -v GITHUB_REPO:$GITHUB_REPO \
    -f manifests/prow/components
```

* Prow dashboard will be accessible at :: `http://<DOMAIN_NAME>`

## Usage
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
    -f manifests/prombench/benchmark
```

### Trigger tests via a Github comment.
---

A Prometheus maintainer can comment as follows to benchmark a PR:
- `/benchmark` (benchmark PR with the master branch.)
- `/benchmark master`
- `/benchmark 2.4.0` (Any release version can be added here. Don't prepend `v` to the release version here. The benchmark plugin in Prow will prepend it.)

To cancel benchmarking, a mantainer should comment `/benchmark cancel`.

## Buliding from source
To build Prombench and related tools from source you need to have a working Go environment with version 1.12 or greater installed. Prombench uses promu for building the binaries.
```
make build
```
