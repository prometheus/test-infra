# test-infra
This repository contains tools and configuration files for the testing and benchmarking used in the Prometheus project.


### [`/prombench`](/prombench)
Prombench is a project for automated E2E testing and benchmarking for Prometheus.

See [prombench/README.md](prombench/README.md) for full description.

### [`/funcbench`](/funcbench)
Funcbench is a project for running `go test -bench` on 2 different branches and showing the difference.

See [funcbench/README.md](funcbench/README.md) for full description.

## Buliding test-infra tools from source
With a working go modules enabled Go environment:
- Install [promu](https://github.com/prometheus/promu): `go install https://github.com/prometheus/promu`
- `promu build`