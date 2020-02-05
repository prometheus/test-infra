DOCKER_REPO             ?= prombench

.PHONY: all
all: precheck style check_license lint build test unused

.PHONY: docker
docker:
	docker build -t "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)" \
		-f $(DOCKERFILE_PATH) $(DOCKERBUILD_CONTEXT)

.PHONY: docker-publish
docker-publish:
	docker push "$(DOCKER_REPO)/$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)"

.PHONY: docker-manifest
docker-manifest:
	@echo skip manifest creation

include Makefile.common
