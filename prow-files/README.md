# How to Deploy Prow

- Create a [ServiceAccount](https://cloud.google.com/kubernetes-engine/docs/tutorials/authenticating-to-cloud-platform#step_3_create_service_account_credentials) on GKE with role `Kubernetes Engine Service Agent & Kubernetes Engine Admin` and download the json file.
- Create a GKE cluster.
```
../prombench gke cluster create -a /etc/serviceaccount/service-account.json -v PROJECT_ID:test \
-v ZONE:us-east1-b -v CLUSTER_NAME:prombench -f cluster.yaml
```
- Initialize kubectl with cluster login credentials
```
gcloud container clusters get-credentials prombench --zone=us-east1-b
```
- Follow [this](https://github.com/kubernetes/test-infra/blob/master/prow/getting_started.md#create-the-github-secrets) to create `hmac-token` and `oauth-token` to talk to GitHub.
```
kubectl create secret generic hmac-token --from-file=hmac=/path/to/hmac-token  
kubectl create secret generic oauth-token --from-file=oauth=/path/to/prom-robot-oauth-token
```
- Add the service-account json file as a kubernetes secret
```
kubectl create secret generic service-account --from-file=service-account.json=/path/to/serviceaccount.json
```
- Create a GCS bucket for [pod-utilities](https://github.com/kubernetes/test-infra/blob/master/prow/pod-utilities.md) using [gsutil](https://cloud.google.com/storage/docs/gsutil_install)
```
gsutil mb -p $PROJECT_ID  gs://prometheus-prow/
```
- Deploy prow components & [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx).
```
../prombench gke resource apply -a /etc/serviceaccount/service-account.json -v PROJECT_ID:test \
-v ZONE:us-east1-b -v CLUSTER_NAME:prombench \
-v GCLOUD_DEVELOPER_ACCOUNT_EMAIL:<client_id in serviceaccount.json> \
-f rbac.yaml -f nginx-controller.yaml -f prow.yaml

kubectl apply -f prowjob.yaml
```
- Run `kubectl get ingress ing` to get ingress IP address. Deploy grafana & prometheus-meta
```
../prombench gke resource apply -a /etc/serviceaccount/service-account.json -v PROJECT_ID:test \
-v ZONE:us-east1-b -v CLUSTER_NAME:prombench -v INGRESS_IP:<Ingress IP address> \
-v GITHUB_ORG:prometheus -v GITHUB_REPO:prometheus \
-v GCS_BUCKET:prometheus-prow \
-f manifests
```
- 
```
The components will be accessible at the following links:
Grafana :: http://INGRESS-IP/grafana
Prometheus ::  http://INGRESS-IP/prometheus-meta
Prow dashboard :: http://INGRESS-IP/
Prow hook :: http://INGRESS-IP/hook
```
(Prow-hook URL should be [added as a webhook](https://github.com/kubernetes/test-infra/blob/master/prow/getting_started.md#add-the-webhook-to-github) in the GitHub repository settings)
- __Don't forget to change Grafana default admin password.__