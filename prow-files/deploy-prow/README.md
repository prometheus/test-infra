# How to Deploy Prow
- Create a GKE cluster.
```
terraform init

terraform apply \
	-var 'region=$REGION' \
	-var 'cluster_name=prombench' \
	-var 'project=$PROJECT_ID' \
	-var 'zone=$ZONE' \
	-var 'kubernetes_version=1.10.4-gke.2'
```

- Get cluster login credentials.
```
gcloud container clusters get-credentials prombench --zone=$ZONE
```
- Follow [this](https://github.com/kubernetes/test-infra/blob/master/prow/getting_started.md#create-the-github-secrets) to create secrets `hmac-token` and `oauth-token` to talk to GitHub.

- Create a [ServiceAccount](https://cloud.google.com/kubernetes-engine/docs/tutorials/authenticating-to-cloud-platform#step_3_create_service_account_credentials) on GKE and add the json file as a kubernetes secret
```
kubectl create secret generic service-account --from-file=service-account.json=<PATH-TO-JSON-KEY-FILE>
```
- Run `kubectl apply -f cluster.yaml` to deploy prow components.

- Run `kubectl get ingress ing` to get ingress IP. 
Update plank.job_url_template in [prow-config.yaml](prow-config.yaml) with ingress IP.

- Create a GCS bucket for [pod-utilities](https://github.com/kubernetes/test-infra/blob/master/prow/pod-utilities.md)
```
gsutil mb -p $PROJECT_ID  gs://prometheus-prow/
```
- Update config configmap
```
kubectl create configmap config --from-file=config=prow-config.yaml --dry-run -o yaml | kubectl replace configmap config -f -
```