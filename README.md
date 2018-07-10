# prometheus-test-environment
An automated E2E testing and benchmarking tool for Prometheus

The design details and instructions on how to run can be found [here](design.md)

Current supports k8s cluster on [GKE - Google Kubernetes Engine](https://cloud.google.com/kubernetes-engine/).

## Build
The project uses [vgo](https://github.com/golang/vgo) without any vendoring.
```
go get -u golang.org/x/vgo
make build // reads go.mod from the project root and downloads all dependancies.
```

## Pre-requisites
1. Create a new GKE project.
2. Create a project service account and save the json file.

## Usage
`gke -h`  - for usage and examples.
