## load-generator
load-generator launches groups of queries against test Prometheus instances in a Prombench test.

### Building Docker Image

From the repository root:

```
make docker DOCKERFILE_PATH=tools/load-generator/Dockerfile DOCKER_IMAGE_NAME=load-generator DOCKER_IMAGE_TAG=master
```
