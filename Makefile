CLUSTER_NAME  ?= prom-test-$(shell whoami)
DOMAIN        ?= dev.coreos.systems
SPEC          ?= spec.example.yaml

path          ?= clusters/${CLUSTER_NAME}
build_path    := $(path)/.build
spec          := $(path)/spec.yaml
aws_region     = $(shell cat $(spec) | yq .awsRegion)

KOPS_CMD        = kops --state $(shell terraform output -state "$(build_path)/terraform.tfstate" kops_state_bucket)
TERRAFORM_FLAGS = -var "dns_domain=$(DOMAIN)" -var "cluster_name=$(CLUSTER_NAME)" -state "$(build_path)/terraform.tfstate" 

MANIFEST_TEMPLATES := $(wildcard manifests/**/*.yaml)
MANIFESTS          := $(patsubst %,$(build_path)/%,$(MANIFEST_TEMPLATES))

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
	AWS_REGION=$(aws_region) terraform apply $(TERRAFORM_FLAGS) ./templates

cluster: manifests aws-deps
	$(KOPS_CMD) get cluster | grep -v $(CLUSTER_NAME).$(DOMAIN) || \
	$(KOPS_CMD) create cluster \
		--name $(CLUSTER_NAME).$(DOMAIN) \
		--cloud aws --zones $(aws_region)a --kubernetes-version 1.5.2 \
		--master-size t2.large --yes
	EDITOR='./ed.sh $(build_path)/manifests/kops/regular-ig.yaml' $(KOPS_CMD) edit ig nodes
	EDITOR='./ed.sh $(build_path)/manifests/kops/prometheus-ig.yaml' $(KOPS_CMD) create ig prometheus
	$(KOPS_CMD) update cluster --yes

cluster-deploy:
	@echo "waiting for cluster to be reachable"
	@until kubectl get nodes; do sleep 15; done
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
