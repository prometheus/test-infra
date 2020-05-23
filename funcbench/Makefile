INFRA_CMD        ?= ../infra/infra

.PHONY: deploy clean
deploy: nodepool_create resource_apply
# GCP sometimes takes longer than 30 tries when trying to delete nodepools
# if k8s resources are not already cleared
clean: resource_delete nodepool_delete

nodepool_create:
	$(INFRA_CMD) gke nodepool create -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/nodepools.yaml

nodepool_delete:
	$(INFRA_CMD) gke nodepool delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/nodepools.yaml

resource_apply:
	$(INFRA_CMD) gke resource apply -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v GITHUB_TOKEN:${GITHUB_TOKEN} \
		-v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
		-v BRANCH:${BRANCH} -v BENCH_FUNC_REGEX:${BENCH_FUNC_REGEX} \
		-f manifests/benchmark

# Removal of namespace should be at the end, after all other resources get removed.
resource_delete:
	$(INFRA_CMD) gke resource delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v GITHUB_TOKEN:${GITHUB_TOKEN} \
		-v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
		-v BRANCH:${BRANCH} -v BENCH_FUNC_REGEX:${BENCH_FUNC_REGEX} \
		-f manifests/benchmark/2_job.yaml \
		-f manifests/benchmark/1_namespace.yaml
