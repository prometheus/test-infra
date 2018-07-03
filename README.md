# A benchmarking tool preconfigured for testing Prometheus
1. Create the k8s cluster described by a configuration file.
2. Apply a k8s resource file to create all required pods,services and config maps.

Current supports k8s cluster on [GKE - Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/).

## Build
The project uses [vgo](https://github.com/golang/vgo) without any vendoring.
```
go get -u golang.org/x/vgo
cd cmd/prombench
vgo build // reads go.mod from the project root and downloads all dependancies.
```

## Pre-requisites
1. Create a new GKE project.
2. Create a project service account and save the json file.

## Usage
`gke -h`  - for usage and examples.
