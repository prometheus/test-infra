#TO_DO 
allow ovverrides from cli
    - MachineType: {{ if .MachineType }} {{ .MachineType }} {{ else }} n1-standard-1 {{ end }}
prombench cluster create --ovverides=MachineType=....,ram=....
use vgo

# prometheus-test-environment
A Kubernetes cluster preconfigured for testing Prometheus


## Pre-requisites
1. Create a GKE project: `prometheus`.
2. Create a project service account and save the json file in `cmd/prombench`.
3. Set the env variable to use the auth file: `export GOOGLE_APPLICATION_CREDENTIALS=key.json`


## Usage
1. Create a cluster using the defaults: 
    ```
    gke cluster create
    ```
2. Delete cluster: 
    ```
    gke cluster delete
    ```
3. Deploy a manifest file:
    ```
    gke deployment apply \
    -f ../../manifests/node-exporter.yaml \
    -f ../../manifests/kube-state-metrics.yaml
    ```

## Kubernetes manifests 
The default Prometheus and it's configuration are store unde `manifests/prometheus`.
Here you can tweak the Prometheus deployment and the configuration file passed to it.