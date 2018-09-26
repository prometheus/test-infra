GO            ?= go
GOFMT         ?= $(GO)fmt
FIRST_GOPATH  := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
PROMU         := $(FIRST_GOPATH)/bin/promu
STATICCHECK   := $(FIRST_GOPATH)/bin/staticcheck
PROMBENCH_CMD := ./prombench
DOCKER_TAG    := docker.io/prombench/prombench:2.0.0
pkgs          := ./...

.PHONY: all
all: style build

.PHONY: style
style: format vet staticcheck

.PHONY: format vet staticcheck build test docker
format:
	@echo ">> formatting code"
	@fmtRes=$$($(GOFMT) -d $$(find . -name '*.go' -print)); \
	if [ -n "$${fmtRes}" ]; then \
		echo "gofmt checking failed!"; echo "$${fmtRes}"; echo; \
		exit 1; \
	fi

vet:
	@echo ">> vetting code"
	$(GO) vet $(pkgs)

staticcheck: $(STATICCHECK)
	@echo ">> running staticcheck"
	$(STATICCHECK) -ignore "$(STATICCHECK_IGNORE)" $(pkgs)

.PHONY: build
build: promu
	@echo ">> building binaries"
	@go version | grep go1.11 || exit  "Requires golang 1.11 with support for modules!"
	@GO111MODULE=on $(PROMU) build

docker: build
	@docker build -t $(DOCKER_TAG) .
	@docker push $(DOCKER_TAG)

.PHONY: $(STATICCHECK)
$(STATICCHECK):
	@GO111MODULE=off GOOS= GOARCH= $(GO) get -u honnef.co/go/tools/cmd/staticcheck

.PHONY: promu
promu:
	@GO111MODULE=off GOOS= GOARCH= $(GO) get -u github.com/prometheus/promu

# Prombench Commands

ifeq ($(AUTH_FILE),)
AUTH_FILE = "/etc/serviceaccount/service-account.json"
endif

.PHONY: deploy clean

deploy: nodepool_create resource_apply

nodepool_create:
	$(PROMBENCH_CMD) gke nodepool create -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f  components/prombench/nodepools.yaml
resource_apply:
	$(PROMBENCH_CMD) gke resource apply -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} \
		-f components/prombench/manifests/benchmark

clean: resource_delete nodepool_delete

resource_delete:
	$(PROMBENCH_CMD) gke resource delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f components/prombench/manifests/benchmark/1a_namespace.yaml -f components/prombench/manifests/benchmark/1c_cluster-role-binding.yaml
nodepool_delete:
	$(PROMBENCH_CMD) gke nodepool delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f components/prombench/nodepools.yaml