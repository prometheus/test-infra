### Prometheus-Builder

This is used for building prometheus binaries from Pull Requests and running them on containers.  
Prombench uses this to build binaries for the Pull Request being benchmarked.

### Building Docker Image

From the repository root:

```
$ make docker DOCKERFILE_PATH=tools/prometheus-builder/Dockerfile DOCKER_IMAGE_NAME=prometheus-builder DOCKER_IMAGE_TAG=master
```
