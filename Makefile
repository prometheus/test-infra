CLUSTER_NAME  ?= prom-test-$(shell whoami)
DOMAIN        ?= monitoring.team.coreos.systems
SPEC          ?= spec.example.yaml

# prometheus test servers + workers + 1x master + 1x prometheus meta server
AMOUNT_NODES	= $$(($(shell cat $(spec) | yq '.prometheus.instances | length')+$(shell cat $(spec) | yq '.workers.count')+1+1))

path          ?= clusters/${CLUSTER_NAME}
build_path    := $(path)/.build
spec          := $(path)/spec.yaml
aws_region     = $(shell cat $(spec) | yq -r .awsRegion)

KOPS_CMD        = kops --state $(shell terraform output -state "$(build_path)/terraform.tfstate" kops_state_bucket)
TERRAFORM_FLAGS = -var "dns_domain=$(DOMAIN)" -var "cluster_name=$(CLUSTER_NAME)" -state "$(build_path)/terraform.tfstate"
MANIFEST_TEMPLATES := $(wildcard manifests/**/*.yaml)
MANIFESTS          := $(patsubst %,$(build_path)/%,$(MANIFEST_TEMPLATES))

master_size = t2.large
node_size = $(shell cat $(spec) | yq -r .workers.machineType)
node_count = $(shell cat $(spec) | yq -r .workers.count)

all: check-deps cluster cluster-deploy

clean: clean-cluster clean-aws-deps clean-manifests

manifests: $(MANIFESTS)

$(spec):
	@mkdir -p $(dir $@)
	@cp $(SPEC) $@

init: $(spec)

$(path)/.build/manifests/%.yaml: init
	@echo "creating manifest $*"
	@mkdir -p $(dir $@)
	@jinja2 manifests/$*.yaml $(spec) > $@

aws-deps:
	AWS_REGION=$(aws_region) terraform init ./templates
	AWS_REGION=$(aws_region) terraform apply $(TERRAFORM_FLAGS) ./templates

cluster: manifests aws-deps
	$(KOPS_CMD) get cluster | grep -v $(CLUSTER_NAME).$(DOMAIN) || \
	$(KOPS_CMD) create cluster \
		--name $(CLUSTER_NAME).$(DOMAIN) \
		--cloud aws --zones $(aws_region)a --kubernetes-version 1.9.2 \
		--master-size $(master_size) --node-size $(node_size) --node-count $(node_count) --yes
	EDITOR='./ed.sh $(build_path)/manifests/kops/regular-ig.yaml' $(KOPS_CMD) edit ig nodes
	EDITOR='./ed.sh $(build_path)/manifests/kops/prometheus-ig.yaml' $(KOPS_CMD) create ig prometheus --subnet $(aws_region)a
	$(KOPS_CMD) update cluster --yes

wait-for-cluster: init
	echo "Going to wait for cluster to become available and nodes to become ready."
	./wait-for-cluster.sh $(AMOUNT_NODES)

cluster-deploy: wait-for-cluster
	kubectl create -f $(build_path)/manifests/k8s

cluster-undeploy:
	kubectl delete -f $(build_path)/manifests/k8s

clean-manifests:
	rm -rf $(build_path)/manifests

clean-cluster:
	$(KOPS_CMD) delete cluster --name $(CLUSTER_NAME).$(DOMAIN) --yes

clean-aws-deps:
	AWS_REGION=$(aws_region) terraform destroy -force $(TERRAFORM_FLAGS) ./templates
	rm -f $(build_path)/terraform.tfstate*

check-deps:
	@which aws || echo "AWS cli is missing. Try to install it with 'brew install awscli'"
	@which kops || echo "Kops is missing. Try to install it with 'brew install kops'"
	@which kubectl || echo "Kubectl is missing. Try to install it with 'brew install kubernetes-cli'"
	@which terraform || echo "Terraform is missing. Try to install it with 'brew install terraform'"
	@which jinja2 || echo "Jinja2 CLI is missing. Try to install with 'pip install pyyaml jinja2-cli[yaml]'"
	@which yq || echo "yq is missing. Try to install with 'pip install yq'"
	@which jq || echo "jq is missing. Try to install with 'pip install jq'"
