SPEC          ?= spec.example.yaml

path          ?= clusters/${CLUSTER_NAME}
build_path    := $(path)/.build
spec          := $(path)/spec.yaml

MANIFEST_TEMPLATES := $(wildcard manifests/*.yaml)
MANIFESTS          := $(patsubst %,$(build_path)/%,$(MANIFEST_TEMPLATES))

deploy: check-deps cluster

clean: clean-manifests 
manifests: $(MANIFESTS)

$(spec):
	@mkdir -p $(dir $@)
	@cp $(SPEC) $@

init: $(spec)

$(path)/.build/manifests/%.yaml: init
	@echo "creating manifest $*"
	@mkdir -p $(dir $@)
	@jinja2 manifests/$*.yaml $(spec) > $@

cluster: manifests

clean-manifests:
	rm -rf $(path) #/manifests

check-deps:
	@which jinja2 || echo "Jinja2 CLI is missing. Try to install with 'pip install pyyaml jinja2-cli[yaml]'"