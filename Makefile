PROMBENCH_CMD := ./prombench
DOCKER_REPO := prombench
DOCKER_IMAGE_NAME := prombench
DOCKER_IMAGE_TAG := 2.0.0

include Makefile.common

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