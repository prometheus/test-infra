# A cli tool to create/scale/delete k8s clusters and deploy manifest files.
Currently it supports GKE, but it is designed in a way that adding more providers should be easy.

### Docker image run instructions
The prombench docker image clones the lastest [prometheus/prombench](https://github.com/prometheus/prombench) and the command to run inside the container can be passed to the `SHELL_COMMAND` env var.
```
docker run -e INIT_EXIT="1" -e SHELL_COMMAND="ls" --rm docker.io/prombench/prombench:2.0.2
```
The prombench container may run multiple processes at once, see [supervisord.ini](./supervisord.ini) for more information.

The `INIT_EXIT` env var is optional, specifying it means container will exit with error code `0` if `init` program exits with an expected error code. Otherwise container will exit based on the `signal-supervisor` event listener.

### Example for building the docker image
From the repository root:
```
$ make docker DOCKERFILE_PATH=prombench/cmd/prombench/Dockerfile DOCKER_IMAGE_NAME=prombench DOCKERBUILD_CONTEXT=prombench/ DOCKER_IMAGE_TAG=2.0.2
```

## Usage and examples:
```
./prombench -h  // Usage and examples.
```
