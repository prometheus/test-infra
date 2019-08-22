# test-infra
This repository contains tools and configuration files for the testing and benchmarking needs of the Prometheus project.


### [`/prombench`](/prombench)
Prombench performs automated E2E testing and benchmarking for Prometheus. It is a set of Kubernetes manifest files, tools and the prombench command line tool itself.

### [`/funcbench`](/funcbench)
A tool used as a GitHub action to run a `go test -bench` and compare changes from a PR against another branch. The benchmark is triggered by creating a comment which specifies a branch to compare. The results are then posted back as a PR comment.

## Buliding test-infra tools from source
To build test-infra related tools from source you need to have a working Go environment with go modules enabled. It uses [promu](https://github.com/prometheus/promu) to build the binaries.
```
make build
```