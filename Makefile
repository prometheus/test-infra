DOCKER_REPO             ?= prombench

include Makefile.common

PROMBENCH_CMD        = ./prombench
KUBECONFIG ?= ${HOME}/.kube/config

ifeq ($(AUTH_FILE),)
AUTH_FILE = "/etc/serviceaccount/service-account.json"
endif

.PHONY: deploy clean
deploy: nodepool_create resource_apply
clean: resource_delete nodepool_delete

nodepool_create:
	$(PROMBENCH_CMD) gke nodepool create -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodepools.yaml

resource_apply:
	$(PROMBENCH_CMD) gke resource apply -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} -v DOMAIN_NAME:${DOMAIN_NAME} \
		-f manifests/prombench/benchmark

resource_delete:
	$(PROMBENCH_CMD) gke resource delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/benchmark/1a_namespace.yaml \
        -f manifests/prombench/benchmark/1c_cluster-role-binding.yaml

nodepool_delete:
	$(PROMBENCH_CMD) gke nodepool delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodepools.yaml

apply_configbootstrapper:
	docker run --rm \
	    -v $(shell pwd):/prombench \
	    -v ${HOME}/.kube:/kube \
	    --network host \
		gcr.io/k8s-prow/config-bootstrapper:v20190608-493ef838c \
	    --dry-run=false \
	    --source-path /prombench \
	    --config-path /prombench/config/prow/config.yaml \
	    --plugin-config /prombench/config/prow/plugins.yaml \
	    --kubeconfig=/kube/$(shell basename $(KUBECONFIG))
