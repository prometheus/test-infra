SPEC          ?= spec.example.yaml

path          ?= clusters/${CLUSTER_NAME}
build_path    := $(path)/.build
spec          := $(path)/spec.yaml

CONFIG_TEMPLATES := $(wildcard config/*.yaml)
CONFIGS          := $(patsubst %,$(build_path)/%,$(CONFIG_TEMPLATES))

MANIFEST_TEMPLATES := $(wildcard manifests/*.yaml)
MANIFESTS          := $(patsubst %,$(build_path)/%,$(MANIFEST_TEMPLATES))

PROMBENCH_CMD        = /bin/prombench
DOCKER_TAG = gcr.io/prometheus-test-204522/prombench:v0.1.0 

deploy: check-deps cluster-deploy

clean: clean-cluster clean-manifests 

build:
	@vgo build -o prombench cmd/prombench/main.go

docker: build
	@docker build -t $(DOCKER_TAG) .
	@docker push $(DOCKER_TAG)

.PHONY: deploy clean build docker

$(spec):
	@mkdir -p $(dir $@)
	@cp $(SPEC) $@

init: $(spec)

$(path)/.build/config/%.yaml: init
	@echo "creating config $*"
	@mkdir -p $(dir $@)
	@jinja2 config/$*.yaml $(spec) > $@

cluster-config: $(CONFIGS)

$(path)/.build/manifests/%.yaml: init
	@echo "creating manifest $*"
	@mkdir -p $(dir $@)
	@jinja2 manifests/$*.yaml $(spec) > $@

manifests: $(MANIFESTS)

.PHONY: init cluster-config manifests

cluster-deploy: cluster-config manifests 
	$(PROMBENCH_CMD) gke cluster create -a /etc/serviceaccount/service-account.json -c $(build_path)/config/cluster.yaml
	$(PROMBENCH_CMD) gke resource apply -a /etc/serviceaccount/service-account.json -c $(build_path)/config/cluster.yaml  -f $(build_path)/manifests

clean-cluster: cluster-config manifests
	$(PROMBENCH_CMD) gke resource delete -a /etc/serviceaccount/service-account.json -c $(build_path)/config/cluster.yaml  -f $(build_path)/manifests
	$(PROMBENCH_CMD) gke cluster delete -a /etc/serviceaccount/service-account.json -c $(build_path)/config/cluster.yaml

clean-manifests:
	rm -rf $(path)
	
check-deps:
	@which jinja2 || echo "Jinja2 CLI is missing. Try to install with 'pip install pyyaml jinja2-cli[yaml]'"

.PHONY: clean-manifests cluster-deploy clean-cluster check-deps
