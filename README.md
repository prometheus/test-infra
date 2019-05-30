# Automated Prometheus E2E testing and benchmarking.

![Prombench Design](design.svg)

It runs with [Prow CI](https://github.com/kubernetes/test-infra/blob/master/prow/) on a [Google Kubernetes Engine Cluster](https://cloud.google.com/kubernetes-engine/).
It is designed to support adding more k8s providers.

## Run tests manually
### Create a k8s cluster
---
- Create a new project on Google Cloud.
- Create a Service Account on GKE with role `Kubernetes Engine Service Agent` & `Kubernetes Engine Admin` and download the json file.

Alternatively you can use the gcloud cli to create a service account:
```
export PROJECT_ID=<google-cloud project-id>
export SERVICE_ACCOUNT_NAME=<gcp-service-account-name>

gcloud iam service-accounts create ${SERVICE_ACCOUNT_NAME} \
  --display-name "prombench service account"
```

Add the `roles/container.admin` and `roles/iam.serviceAccountUser` roles to the GKE serviceAccount:

```
gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role='roles/container.admin'

gcloud projects add-iam-policy-binding ${PROJECT_ID} \
  --member="serviceAccount:${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  --role='roles/iam.serviceAccountUser'
```

Get the serviceAccount key:
```
gcloud iam service-accounts keys create \
  --iam-account "${SERVICE_ACCOUNT_NAME}@${PROJECT_ID}.iam.gserviceaccount.com" \
  service-account.json
```

- Set the following environment variables & deploy cluster.
```
export PROJECT_ID=<google-cloud project-id>
export CLUSTER_NAME=prombench
export ZONE=us-east1-b
export AUTH_FILE=<path to service-account.json>

./prombench gke cluster create -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID \
    -v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME -f components/prow/cluster.yaml
```

### Deploy Prometheus-Meta & Grafana
> This is used for collecting and displaying the test results.

---

- Set the following environment variables
```
export GCLOUD_SERVICEACCOUNT_CLIENTID=<client-id present in service-account.json>
export GRAFANA_ADMIN_PASSWORD=password
```
> The `GCLOUD_SERVICEACCOUNT_CLIENTID` is used to grant cluster-admin-rights to the service-account. This is needed to create RBAC roles on GKE.

- Deploy the [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx) which will be used to access Prometheus-Meta & Grafana.
```
./prombench gke resource apply -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID -v ZONE:$ZONE \
    -v CLUSTER_NAME:$CLUSTER_NAME -v GCLOUD_SERVICEACCOUNT_CLIENTID:$GCLOUD_SERVICEACCOUNT_CLIENTID \
    -f components/prow/manifests/rbac.yaml -f components/prow/manifests/nginx-controller.yaml
```

- Export the nginx-ingress-controller IP address.
```
// Generate auth config so we can use kubectl.
gcloud container clusters get-credentials $CLUSTER_NAME --zone=$ZONE --project=$PROJECT_ID

kubectl get ingress ing -o go-template='{{ range .status.loadBalancer.ingress}}{{.ip}}{{ end }}'

export INGRESS_IP=$(kubectl get ingress ing -o go-template='{{ range .status.loadBalancer.ingress}}{{.ip}}{{ end }}')
```

- Deploy Prometheus-meta & Grafana.
```
./prombench gke resource apply -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID \
    -v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME -v INGRESS_IP:$INGRESS_IP \
    -v GRAFANA_ADMIN_PASSWORD:$GRAFANA_ADMIN_PASSWORD -f components/prombench/manifests/results
```

- The services will be accessible at:
  * Grafana :: http://<INGRESS_IP>/grafana
  * Prometheus :: http://<INGRESS_IP>/prometheus-meta

### Start a test
---

- Set the following environment variables.
```
export RELEASE=<master or any prometheus release(ex: v2.3.0) >
export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
```

- Create the nodepools for the k8s objects
```
./prombench gke nodepool create -a ${AUTH_FILE} \
    -v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
    -v PR_NUMBER:${PR_NUMBER} -f components/prombench/nodepools.yaml
```

- Deploy the k8s objects
```
./prombench gke resource apply -a ${AUTH_FILE} \
    -v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
    -v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} \
    -f components/prombench/manifests/benchmark
```

## Triggered tests by GitHub comments

### Create a k8s cluster
---

- Follow the steps mentioned in the [Create a k8s cluster](#create-a-k8s-cluster) in the manual setup.

### Setup the GitHub API
---

- Generate a GitHub auth token that will be used to authenticate when sending requests to the GitHub api.
  * Login with the [Prombot account](https://github.com/prombot) and generate a [new auth token](https://github.com/settings/tokens).  
  permissions:*public_repo, read:org, write:discussion*.

- Set the following environment variables
```
export HMAC_TOKEN=$(openssl rand -hex 20)
export OAUTH_TOKEN=***Replace with the generated token from github***
```

- Add a [github webhook](https://github.com/prometheus/prometheus/settings/hooks) where to send the events.
  * Content Type: `json`
  * Send:  `Issue comments,Pull requests`
  * Secret: `echo $HMAC_TOKEN`
  * Payload URL: `http://prombench.prometheus.io/hook`

    * **Note:** The IP DNS record for `prombench.prometheus.io` will be added once we get it from the ingress deployment.

### Deploy Prow
> This is used to monitor GitHub comments and starts new tests.

---

- Add all required tokens as k8s secrets.
  * hmac is used when verifying requests from GitHub.
  * oauth is used when sending requests to the GitHub api.
  * gke auth is used when scaling up and down the cluster.
```
./prombench gke resource apply -a $AUTH_FILE -v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME \
    -f components/prow/manifests/secrets.yaml \
    -v HMAC_TOKEN="$(printf $HMAC_TOKEN | base64 -w 0)" \
    -v OAUTH_TOKEN="$(printf $OAUTH_TOKEN | base64 -w 0)" \
    -v GKE_AUTH="$(cat $AUTH_FILE | base64 -w 0)"
```

- Deploy all internal prow components

> Long term plans are to use the [prombench cli tool](cmd/prombench) to deploy and manage everything, but at the moment `CustomResourceDefinition` is WIP in the k8s golang client library. So we use `kubectl` to deploy CRD.
```
// Generate auth config so we can use kubectl.
gcloud container clusters get-credentials $CLUSTER_NAME --zone=$ZONE
kubectl apply -f components/prow/manifests/prow_internals_1.yaml

export GITHUB_ORG=prometheus
export GITHUB_REPO=prometheus

./prombench gke resource apply -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID \
    -v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME \
    -v GITHUB_ORG:$GITHUB_ORG -v GITHUB_REPO:$GITHUB_REPO \
    -f components/prow/manifests/prow_internals_2.yaml
```

### Deploy Prometheus-Meta & Grafana
---
- Follow the steps mentioned in the [manual](#deploy-prometheus-meta--grafana) setup to deploy prometheus-meta & grafana.

- Set the IP DNS record for `prombench.prometheus.io` to the nginx-ingress-controller IP address.

- The services will be accessible at:
  * Prow dashboard :: http://prombench.prometheus.io
  * Grafana :: http://prombench.prometheus.io/grafana
  * Prometheus ::  http://prombench.prometheus.io/prometheus-meta

### Trigger tests via a Github comment.
---

A Prometheus maintainer can comment as follows to benchmark a PR:
- `/benchmark` (benchmark PR with the master branch.)
- `/benchmark master`
- `/benchmark 2.4.0` (Any release version can be added here. Don't prepend `v` to the release version here. The benchmark plugin in Prow will prepend it.)

To cancel benchmarking, a mantainer should comment `/benchmark cancel`.
