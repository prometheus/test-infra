REGION?=eu-west-1
CLUSTER_NAME?=prom-test-$(shell whoami)
DOMAIN?=dev.coreos.systems
KOPS_CMD=kops --state $(shell terraform output kops_state_bucket)

all: check-deps cluster

clean: clean-cluster clean-aws-deps

aws-deps:
	AWS_REGION=$(REGION) terraform apply -var "dns_domain=$(DOMAIN)" -var "cluster_name=$(CLUSTER_NAME)" ./templates

cluster: aws-deps
	$(KOPS_CMD) get cluster | grep -v $(CLUSTER_NAME).$(DOMAIN) || \
	$(KOPS_CMD) create cluster --name $(CLUSTER_NAME).$(DOMAIN) \
		--cloud aws --zones $(REGION)a
	EDITOR='./ed.sh manifests/kops/regular-ig.yaml' $(KOPS_CMD) edit ig nodes
	EDITOR='./ed.sh manifests/kops/prometheus-ig.yaml' $(KOPS_CMD) create ig prometheus
	$(KOPS_CMD) update cluster --yes

cluster-deploy: cluster
	kubectl create -f ./manifests/exporters
	kubectl create -f ./manifests/prometheus

cluster-undeploy:
	kubectl delete -f ./manifests/prometheus

clean-cluster:
	$(KOPS_CMD) delete cluster --name $(CLUSTER_NAME).$(DOMAIN) --yes

clean-aws-deps: clean-cluster
	AWS_REGION=$(REGION) terraform destroy -force -var "dns_domain=$(DOMAIN)" -var "cluster_name=$(CLUSTER_NAME)" ./templates
	rm -f terraform.tfstate*

check-deps:
	@which aws || echo "AWS cli is missing. Try to install it with 'brew install awscli'"
	@which kops || echo "Kops is missing. Try to install it with 'brew install kops'"
	@which kubectl || echo "Kubectl is missing. Try to install it with 'brew install kubernetes-cli'"
	@which terraform || echo "Terraform is missing. Try to install it with 'brew install terraform'"
