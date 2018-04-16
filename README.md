#TO_DO 

- can we simplify update if exists , create if doesn't
- Service update doesn't work!
- use more sane logging that shows the line in the code of the log  and show info logs only when debug mode is enabled.

# A Kubernetes cluster preconfigured for testing Prometheus
currently supports creating and using a k8s cluster on [GKE - Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/)

## Build
The project uses [vgo](https://github.com/golang/vgo) without any vendoring.
```
go get -u golang.org/x/vgo
cd cmd/prombench
vgo build // This will read go.mod from project root and download all dependancies.
```

## Pre-requisites
1. Create a new GKE project.
2. Create a project service account and save the json file in `cmd/prombench`.

## Usage
`gke -h`  - for usage and examples
