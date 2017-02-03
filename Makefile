AWS_REGION    ?= eu-west-1
CLUSTER_NAME  ?= prom-test-$(shell whoami)
DOMAIN        ?= dev.coreos.systems
KOPS_CMD       = kops --state $(shell terraform output kops_state_bucket)

path ?= clusters/${CLUSTER_NAME}

NUM_WORKERS         ?= 3
WORKER_MACHINE_TYPE ?= t2.medium
PROM_MACHINE_TYPE   ?= c3.2xlarge

KOPS_TEMPLATES = $(patsubst %.yaml,$(path)/.build/manifests/kops/%.yaml,regular-ig.yaml prometheus-ig.yaml)

MANIFEST_TEMPLATES := $(wildcard manifests/**/*.yaml)
MANIFESTS          := $(patsubst %,$(path)/.build/%,$(MANIFEST_TEMPLATES))

TEMPLATE_VARS = \
	AWS_REGION=$(AWS_REGION) \
	NUM_WORKERS=$(NUM_WORKERS) \
	WORKER_MACHINE_TYPE=$(WORKER_MACHINE_TYPE) \
	PROM_MACHINE_TYPE=$(PROM_MACHINE_TYPE)


all: check-deps cluster

clean: clean-cluster clean-aws-deps clean-manifests

aws-deps:
	AWS_REGION=$(AWS_REGION) terraform apply -var "dns_domain=$(DOMAIN)" -var "cluster_name=$(CLUSTER_NAME)" ./templates

cluster: aws-deps manifests
	$(KOPS_CMD) get cluster | grep -v $(CLUSTER_NAME).$(DOMAIN) || \
	$(KOPS_CMD) create cluster \
		--name $(CLUSTER_NAME).$(DOMAIN) \
		--cloud aws --zones $(AWS_REGION)a --kubernetes-version 1.5.2 \
		--master-size t2.large --yes
	EDITOR='./ed.sh $(path)/.build/manifests/kops/regular-ig.yaml' $(KOPS_CMD) edit ig nodes
	EDITOR='./ed.sh $(path)/.build/manifests/kops/prometheus-ig.yaml' $(KOPS_CMD) create ig prometheus
	$(KOPS_CMD) update cluster --yes

manifests: $(MANIFESTS)

$(path)/.build/manifests/%.yaml:
	mkdir -p $(dir $@)
	cat manifests/$*.yaml | $(TEMPLATE_VARS) envsubst > $@

cluster-deploy:
	kubectl create -f ./manifests/prometheus

cluster-undeploy:
	kubectl delete -f ./manifests/prometheus

clean-manifests:
	rm -rf $(path)/.build/manifests

clean-cluster:
	$(KOPS_CMD) delete cluster --name $(CLUSTER_NAME).$(DOMAIN) --yes

clean-aws-deps:
	AWS_REGION=$(AWS_REGION) terraform destroy -force -var "dns_domain=$(DOMAIN)" -var "cluster_name=$(CLUSTER_NAME)" ./templates
	rm -f terraform.tfstate*

check-deps:
	@which aws || echo "AWS cli is missing. Try to install it with 'brew install awscli'"
	@which kops || echo "Kops is missing. Try to install it with 'brew install kops'"
	@which kubectl || echo "Kubectl is missing. Try to install it with 'brew install kubernetes-cli'"
	@which terraform || echo "Terraform is missing. Try to install it with 'brew install terraform'"
