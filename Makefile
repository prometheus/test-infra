DOCKER_REPO             ?= prominfra

.PHONY: all
all: precheck style check_license lint build test unused docs-check generate-dashboards-cm

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

.PHONY: docs
docs:
	./scripts/genflagdocs.sh

.PHONY: docs-check
docs-check:
	./scripts/genflagdocs.sh check

.PHONY: generate-dashboards-cm
generate-dashboards-cm:
	./scripts/sync-dashboards-to-configmap.sh

include Makefile.common
