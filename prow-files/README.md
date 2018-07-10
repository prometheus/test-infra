# How to Deploy Prow

- Create a [ServiceAccount](https://cloud.google.com/kubernetes-engine/docs/tutorials/authenticating-to-cloud-platform#step_3_create_service_account_credentials) on GKE with role `Kubernetes Engine Service Agent & Kubernetes Engine Admin` and download the json file.

- Create a GKE cluster.
```
../prombench gke cluster create -a /etc/serviceaccount/service-account.json -c cluster.yaml
```

- Initialize kubectl with cluster login credentials
```
gcloud container clusters get-credentials prombench --zone=$ZONE
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

- Since GKE requires cluster-admin-binding to be granted to user before allowing to create RBAC rules, add the following email-ids as User in [rbac.yaml](rbac.yaml)
	- email associated with gcloud developer account (`gcloud config get-value account`)

- Run `kubectl apply -f rbac.yaml && kubectl apply -f nginx-controller.yaml && kubectl apply -f prow.yaml` to deploy prow components & [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx).

- Run `kubectl get ingress ing` to get ingress IP address. This step can take some time to complete.

- Replace INGRESS-IP in [prow-config.yaml#L8](manifests/prow-config.yaml#L8), [grafana.yaml#L34](manifests/grafana.yaml#L34) & [grafana_datasource.yaml#L15](manifests/grafana_datasource.yaml#L15) with the ingress IP address. You can also use `sed -i 's/INGRESS-IP/35.190.88.109/g' manifests/*` to replace these values 

- Add your organization & repository name in [prow-config.yaml#L21](manifests/prow-config.yaml#L21) & [prow-config.yaml#L39](manifests/prow-config.yaml#L39)

- Add the GCS bucket name in [prow-config.yaml#L19](manifests/prow-config.yaml#L19) & the cluster-name & zone in the env variables in 
[prow-config.yaml](manifests/prow-config.yaml)

- Deploy Prometheus, Grafana & prow config
```
kubectl apply -f manifests

The components will be accessible at the following links:
Grafana :: http://INGRESS-IP/grafana
Prometheus ::  http://INGRESS-IP/prometheus-meta
Prow dashboard :: http://INGRESS-IP/
Prow hook :: http://INGRESS-IP/hook
(The hook URL will be added as a webhook in the GitHub repository settings)
```
- __Don't forget to change Grafana default admin password.__