SPEC          ?= spec.example.yaml

path          ?= clusters/${CLUSTER_NAME}
build_path    := $(path)/.build
spec          := $(path)/spec.yaml

CONFIG_TEMPLATES := $(wildcard config/*.yaml)
CONFIGS          := $(patsubst %,$(build_path)/%,$(CONFIG_TEMPLATES))

MANIFEST_TEMPLATES := $(wildcard manifests/*.yaml)
MANIFESTS          := $(patsubst %,$(build_path)/%,$(MANIFEST_TEMPLATES))

deploy: check-deps cluster

clean: clean-manifests

$(spec):
	@mkdir -p $(dir $@)
	@cp $(SPEC) $@

init: $(spec)

$(path)/.build/config/%.yaml: init
	@echo "creating config $*"
	@mkdir -p $(dir $@)
	@jinja2 config/$*.yaml $(spec) > $@
cluster-yaml: $(CONFIGS)

$(path)/.build/manifests/%.yaml: init
	@echo "creating manifest $*"
	@mkdir -p $(dir $@)
	@jinja2 manifests/$*.yaml $(spec) > $@
manifests-yaml: $(MANIFESTS)

cluster: manifests-yaml cluster-yaml

clean-manifests:
	rm -rf $(path)

check-deps:
	@which jinja2 || echo "Jinja2 CLI is missing. Try to install with 'pip install pyyaml jinja2-cli[yaml]'"