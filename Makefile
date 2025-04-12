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

GOIMPORTS = goimports
$(GOIMPORTS):
	@go install golang.org/x/tools/cmd/goimports@latest

GOFUMPT = gofumpt
$(GOFUMPT):
	@go install mvdan.cc/gofumpt@latest

GO_FILES = $(shell find . -path ./vendor -prune -o -name '*.go' -print)

.PHONY: format
format: $(GOFUMPT) $(GOIMPORTS)
	@echo ">> formating imports)"
	@$(GOIMPORTS) -local github.com/prometheus/test-infra -w $(GO_FILES)
	@echo ">> gofumpt-ing the code; golangci-lint requires this"
	@$(GOFUMPT) -extra -w $(GO_FILES)

include Makefile.common
