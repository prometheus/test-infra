PROMBENCH_CMD        = ./prombench

ifeq ($(AUTH_FILE),)
AUTH_FILE = /etc/serviceaccount/service-account.json
endif

.PHONY: deploy clean
deploy: nodepool_create resource_apply
# GCP sometimes takes longer than 30 tries when trying to delete nodepools
# if k8s resources are not already cleared
clean: resource_delete nodepool_delete

nodepool_create:
	$(PROMBENCH_CMD) gke nodepool create -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodepools.yaml

resource_apply:
	$(PROMBENCH_CMD) gke resource apply -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} -v DOMAIN_NAME:${DOMAIN_NAME} \
		-v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
		-f manifests/prombench/benchmark

# NOTE: required because namespace and cluster-role are not part of the created nodepools
resource_delete:
	$(PROMBENCH_CMD) gke resource delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/benchmark/1c_cluster-role-binding.yaml \
		-f manifests/prombench/benchmark/1a_namespace.yaml

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

# funcbench resource
funcbench_nodepool_create:
	$(PROMBENCH_CMD) gke nodepool create -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/funcbench/nodepools.yaml

funcbench_nodepool_delete:
	$(PROMBENCH_CMD) gke nodepool delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/funcbench/nodepools.yaml

funcbench_resource_apply:
	$(PROMBENCH_CMD) gke resource apply -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v GITHUB_TOKEN:${GITHUB_TOKEN} \
		-v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
		-v BRANCH:${BRANCH} -v BENCH_FUNC_REGEX:${BENCH_FUNC_REGEX} \
		-f manifests/funcbench/benchmark

# Removal of namespace should be at the end, after all other resources get removed.
funcbench_resource_delete:
	$(PROMBENCH_CMD) gke resource delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v GITHUB_TOKEN:${GITHUB_TOKEN} \
		-v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
		-v BRANCH:${BRANCH} -v BENCH_FUNC_REGEX:${BENCH_FUNC_REGEX} \
		-f manifests/funcbench/benchmark/2_job.yaml \
		-f manifests/funcbench/benchmark/1_namespace.yaml
