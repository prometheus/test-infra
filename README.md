# Automated E2E testing and benchmarking tool for Prometheus.


### The overall components and intercation is shown in this diagram:

![Prombench Design](design.svg)

It runs with Prow CI on a [GKE - Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/) k8s cluster and it is designed in a way to support adding more k8s providers.

Long term plans are to use the [prombench cli tool](https://github.com/prometheus/prombench/tree/master/cmd/prombench) to deploy and manage everything, but at the moment the  k8s golang client doesn't support `CustomResourceDefinition` objects so for those it uses `kubectl`.

## Prerequisites 
- Create a new Google cloud project - `prometheus-ci`
- Create a [Service Account](https://cloud.google.com/kubernetes-engine/docs/tutorials/authenticating-to-cloud-platform#step_3_create_service_account_credentials) on GKE with role `Kubernetes Engine Service Agent & Kubernetes Engine Admin` and download the json file.
- [Create a GCS bucket](https://console.cloud.google.com/storage/) `prow` in the same project which will be used for [pod-utilities](https://github.com/kubernetes/test-infra/blob/master/prow/pod-utilities.md)

- Set some env variable which will be used in different commands.
```
export PROJECT_ID=prometheus-ci 
export CLUSTER_NAME=prow
export ZONE=us-east1-b
export GCLOUD_DEVELOPER_ACCOUNT_EMAIL=<client_email from the service-account.json>
export AUTH_FILE=<path to service-account.json>
export GCS_BUCKET=prow
export GITHUB_ORG=prometheus
export GITHUB_REPO=prometheus
```
## Setup Prow CI

- Create the main k8s cluster to deploy the Prow components.

```
../prombench gke cluster create -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID \
-v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME -f prow.yaml
```

## Initialize kubectl with cluster login credentials
```
gcloud container clusters get-credentials $CLUSTER_NAME --zone=$ZONE
```
## Follow [this](https://github.com/kubernetes/test-infra/blob/master/prow/getting_started.md#create-the-github-secrets) to create `hmac-token` and `oauth-token` to talk to GitHub.
```
kubectl create secret generic hmac-token --from-file=hmac=/path/to/hmac-token  
kubectl create secret generic oauth-token --from-file=oauth=/path/to/prom-robot-oauth-token
```
## Add the service-account json file as a kubernetes secret
```
kubectl create secret generic service-account --from-file=service-account.json=$AUTH_FILE
```

## Deploy all internal prow components and the [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx) which will be used to access all public components.
```
../prombench gke resource apply -a $AUTH_FILE -v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME \
-v GCLOUD_DEVELOPER_ACCOUNT_EMAIL:$GCLOUD_DEVELOPER_ACCOUNT_EMAIL \
-f rbac.yaml -f nginx-controller.yaml -f prow.yaml

export INGRESS_IP=$(kubectl get ingress ingress-nginx)

kubectl apply -f prowjob.yaml
```

## Deploy grafana & prometheus-meta.
```
../prombench gke resource apply -a $AUTH_FILE -v PROJECT_ID:$PROJECT_ID \
-v ZONE:$ZONE -v CLUSTER_NAME:$CLUSTER_NAME -v INGRESS_IP:$INGRESS_IP \
-v GITHUB_ORG:$GITHUB_ORG -v GITHUB_REPO:$GITHUB_ORG \
-v GCS_BUCKET:$GCS_BUCKET \
-f manifests
```

The components will be accessible at the following links:
  * Grafana ::  http://INGRESS-IP/grafana
  * Prometheus ::  http://INGRESS-IP/prometheus-meta
  * Prow dashboard :: http://INGRESS-IP/
  * Prow hook :: http://INGRESS-IP/hook

(Prow-hook URL should be [added as a webhook](https://github.com/kubernetes/test-infra/blob/master/prow/getting_started.md#add-the-webhook-to-github) in the GitHub repository settings)
- __Don't forget to change Grafana default admin password.__