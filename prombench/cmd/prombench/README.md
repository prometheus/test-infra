# A cli tool to create/scale/delete k8s clusters and deploy manifest files.
Currently it supports GKE, but it is designed in a way that adding more providers should be easy.

### Example for building the docker image
From the repository root:
```
$ make docker DOCKERFILE_PATH=prombench/cmd/prombench/Dockerfile DOCKER_IMAGE_NAME=prombench 
DOCKERBUILD_CONTEXT=prombench/ DOCKER_IMAGE_TAG=2.0.2
```

## Usage and examples:
```
./prombench -h  // Usage and examples.
```
