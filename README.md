# test-infra
This repository contains tools and configuration files for the testing and benchmarking needs of the Prometheus project.

## Projects 

#### Prombench
Prombench performs automated Prometheus E2E testing and benchmarking it is a set of Kubernetes manifest files, tools and the prombench command line tool itself.

## Buliding test-infra tools from source
To build test-infra related tools from source you need to have a working Go environment with go modules enabled. It uses [promu](https://github.com/prometheus/promu) to build the binaries.
```
make build
```