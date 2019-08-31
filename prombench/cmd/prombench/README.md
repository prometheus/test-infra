# A cli tool to create/scale/delete k8s clusters and deploy manifest files.
Currently it supports GKE, but it is designed in a way that adding more providers should be easy.

### Docker image run instructions
The prombench docker image clones the lastest [prometheus/prombench](https://github.com/prometheus/prombench) and the command to run inside the container can be passed to the `SHELL_COMMAND` env var.
```
docker run -e SHELL_COMMAND="ls" --rm docker.io/prombench/prombench:2.0.2
```
The prombench container may run multiple processes at once, see [supervisord.conf](./supervisord.conf) for more information.

### Example for building the docker image
From the repository root:
```
$ make docker DOCKERFILE_PATH=prombench/cmd/prombench/Dockerfile DOCKER_IMAGE_NAME=prombench DOCKERBUILD_CONTEXT=prombench/ DOCKER_IMAGE_TAG=2.0.2
```

## Usage and examples:
```
./prombench -h  // Usage and examples.
```
