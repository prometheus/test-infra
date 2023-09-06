# Prombench in GKE

Run prombench tests in [Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine).

## Setup prombench

1. [Create the main node](#create-the-main-node)
2. [Deploy monitoring components](#deploy-monitoring-components)
3. [Start a benchmarking test manually](#start-a-benchmarking-test-manually)

### Create the Main Node

---

- Create a new project on Google Cloud.
- Create a [Service Account](https://cloud.google.com/iam/docs/creating-managing-service-accounts) on GKE with role `Kubernetes Engine Service Agent` & `Kubernetes Engine Admin`. If using gcloud cli add the [`roles/container.admin`](https://cloud.google.com/kubernetes-engine/docs/how-to/iam#kubernetes-engine-roles) and [`roles/iam.serviceAccountUser`](https://cloud.google.com/kubernetes-engine/docs/how-to/iam#service_account_user) roles to the GCP serviceAccount and download the json file.

- Set the following environment variables and deploy the cluster.

```
export GKE_PROJECT_ID=<google-cloud project-id>
export CLUSTER_NAME=prombench
export ZONE=us-east1-b
export AUTH_FILE=<path to service-account.json>
export PROVIDER=gke

make cluster_create
```

### Deploy monitoring components

> Collecting, monitoring and displaying the test results and logs

---

- [Optional] If used with the Github integration generate a GitHub auth token.
  - Login with the [Prombot account](https://github.com/prombot) and generate a [new auth token](https://github.com/settings/tokens).
  - With permissions: `public_repo`, `read:org`, `write:discussion`.

```
export SERVICEACCOUNT_CLIENT_EMAIL=<client-email present in service-account.json>
export GRAFANA_ADMIN_PASSWORD=password
export DOMAIN_NAME=prombench.prometheus.io # Can be set to any other custom domain or an empty string when not used with the Github integration.
export OAUTH_TOKEN=<generated token from github or set to an empty string " ">
export WH_SECRET=<github webhook secret>
export GITHUB_ORG=prometheus
export GITHUB_REPO=prometheus
```

- Deploy the [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx), Prometheus-Meta, Loki, Grafana, Alertmanager & Github Notifier.

```
make cluster_resource_apply
```

- The output will show the ingress IP which will be used to point the domain name to. Alternatively you can see it from the GKE/Services tab.
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

```
export RELEASE=<master/main or any prometheus release(ex: v2.3.0) >
export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
```

- Create the nodepools for the k8s objects

```
make node_create
```

- Deploy the k8s objects

```
make resource_apply
```

### Stopping a benchmarking test manually

---

- Set the following environment variables:
```
export GKE_PROJECT_ID=<google-cloud project-id>
export CLUSTER_NAME=prombench
export ZONE=us-east1-b
export AUTH_FILE=<path to service-account.json>
export PROVIDER=gke

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
