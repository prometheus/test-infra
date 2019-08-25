### Prometheus-Builder

This is used for building prometheus binaries from Pull Requests and running them on containers.  
Prombench uses this to build binaries for the Pull Request being benchmarked.

### Example for building the docker image
From the repository root:
```
$ make docker DOCKERFILE_PATH=prombench/temp/prometheus-builder/Dockerfile DOCKERBUILD_CONTEXT=prombench/temp/prometheus-builder DOCKER_IMAGE_NAME=prometheus-builder DOCKER_IMAGE_TAG=2.0.2
```
