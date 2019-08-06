DOCKER_REPO             ?= prombench

include Makefile.common

PROMBENCH_CMD        = ./prombench

ifeq ($(AUTH_FILE),)
AUTH_FILE = /etc/serviceaccount/service-account.json
endif

ifdef PULL_PULL_SHA
GITHUB_SHA = $(PULL_PULL_SHA)
endif

.PHONY: deploy clean
deploy: nodepool_create resource_apply
clean: nodepool_delete resource_delete

start_ss:
	$(PROMBENCH_CMD) gke resource apply -a ${AUTH_FILE} -v PROJECT_ID:${PROJECT_ID} \
		-v ZONE:${ZONE} -v CLUSTER_NAME:${CLUSTER_NAME} -v DOMAIN_NAME:${DOMAIN_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} -v LAST_COMMIT:${GITHUB_SHA} \
		-v PROMBENCH_REPO:${PROMBENCH_REPO} \
		-f manifests/prombench/stateful-set.yaml

stop_ss:
	$(PROMBENCH_CMD) gke resource delete -a ${AUTH_FILE} -v PROJECT_ID:${PROJECT_ID} \
		-v ZONE:${ZONE} -v CLUSTER_NAME:${CLUSTER_NAME} -v DOMAIN_NAME:${DOMAIN_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} -v LAST_COMMIT:${GITHUB_SHA} \
		-v PROMBENCH_REPO:${PROMBENCH_REPO} \
		-f manifests/prombench/stateful-set.yaml

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

all_nodepools_running:
	$(PROMBENCH_CMD) gke nodepool check-running -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodepools.yaml

all_nodepools_deleted:
	$(PROMBENCH_CMD) gke nodepool check-deleted -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodepools.yaml
