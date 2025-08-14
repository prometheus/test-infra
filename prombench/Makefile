INFRA_CMD  ?= ../infra/infra
PROVIDER   ?= gke

cluster_create:
	${INFRA_CMD} ${PROVIDER} cluster create -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} -v SEPARATOR:${SEPARATOR} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/cluster_${PROVIDER}.yaml

cluster_resource_apply:
	${INFRA_CMD} ${PROVIDER} resource apply -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} -v SEPARATOR:${SEPARATOR} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} -v DOMAIN_NAME:${DOMAIN_NAME} -v RELEASE:${RELEASE} \
		-v GRAFANA_ADMIN_PASSWORD:${GRAFANA_ADMIN_PASSWORD} \
		-v SERVICEACCOUNT_CLIENT_EMAIL:${SERVICEACCOUNT_CLIENT_EMAIL} \
		-v OAUTH_TOKEN="$(printf ${OAUTH_TOKEN} | base64 -w 0)" \
		-v WH_SECRET="$(printf ${WH_SECRET} | base64 -w 0)" \
		-v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
		-f manifests/cluster-infra

cluster_delete:
	${INFRA_CMD} ${PROVIDER} cluster delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} -v SEPARATOR:${SEPARATOR} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/cluster_${PROVIDER}.yaml

# block_sync_setup allows the creation of the prombench-${PR_NUMBER} namespace using the infra CMD.
# It also copies the blocksync-configmap and bucket-secret from the default namespace to the prombench-${PR_NUMBER} namespace.
# This is necessary because block-sync requires these two resources to download data from the object storage.
block_sync_setup:
	${INFRA_CMD} ${PROVIDER} copy -v PR_NUMBER:${PR_NUMBER}

# /prombench <...> --bench.directory
BENCHMARK_DIRECTORY := $(if $(BENCHMARK_DIRECTORY),$(BENCHMARK_DIRECTORY),manifests/prombench)
# /prombench <...> --bench.version
BENCHMARK_VERSION := $(if $(BENCHMARK_VERSION),$(BENCHMARK_VERSION),master)
PROMBENCH_GIT_REPOSITORY ?= https://github.com/prometheus/test-infra.git
PROMBENCH_DIR ?= .

# maybe_pull_custom_version allows custom benchmarking as designed in
# https://github.com/prometheus/proposals/pull/41. It allows calling
# /prombench <release> --bench.version=<@commit or branch> which will cause
# prombench GH job on Prometheus repo to call infra CLI with the non-master BENCHMARK_VERSION.
# In such a case we pull a prombench repository for the given branch or commit version
# and adjust PROMBENCH_DIR. As a result `make deploy` and `make clean` jobs
# will apply /manifests/ apply custom manifests or even node pools.
.PHONY: maybe_pull_custom_version
maybe_pull_custom_version:
ifeq (${BENCHMARK_VERSION},master)
	@echo ">> Using standard benchmark configuration, from the docker image"
else
	@echo ">> Git pulling custom benchmark configuration from the ${BENCHMARK_VERSION}"
	@$(eval $@_TMP_DIR=$(shell mktemp -d))
	cd ${$@_TMP_DIR} && git clone ${PROMBENCH_GIT_REPOSITORY}
ifeq ($(subst @,,${BENCHMARK_VERSION}),${BENCHMARK_VERSION})
	@echo ">> --bench.version is a branch, reseting to origin/${BENCHMARK_VERSION}"
	cd ${$@_TMP_DIR}/test-infra && git reset --hard origin/${BENCHMARK_VERSION}
else
	@echo ">> --bench.version is a commit SHA, reseting to $(subst @,,${BENCHMARK_VERSION})"
	cd ${$@_TMP_DIR}/test-infra && git reset --hard $(subst @,,${BENCHMARK_VERSION})
endif
	$(eval PROMBENCH_DIR=${$@_TMP_DIR}/test-infra/prombench)
endif
	@echo ">> Using following files in ${PROMBENCH_DIR}/${BENCHMARK_DIRECTORY}"
	@ls -lR ${PROMBENCH_DIR}/${BENCHMARK_DIRECTORY}

.PHONY: clean_tmp_dir
clean_tmp_dir: # Clean after maybe_pull_custom_version
	[ -z ${maybe_pull_custom_version_TMP_DIR} ] || rm -rf ${maybe_pull_custom_version_TMP_DIR}

.PHONY: deploy
deploy: maybe_pull_custom_version node_create block_sync_setup resource_apply clean_tmp_dir

.PHONY: clean
# GCP sometimes takes longer than 30 tries when trying to delete nodes
# if k8s resources are not already cleared
clean: maybe_pull_custom_version resource_delete node_delete clean_tmp_dir

node_create:
	${INFRA_CMD} ${PROVIDER} nodes create -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f ${PROMBENCH_DIR}/${BENCHMARK_DIRECTORY}/nodes_${PROVIDER}.yaml

resource_apply:
	$(INFRA_CMD) ${PROVIDER} resource apply -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} -v DOMAIN_NAME:${DOMAIN_NAME} \
		-v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
		-f ${PROMBENCH_DIR}/${BENCHMARK_DIRECTORY}/benchmark

# Required because namespace and cluster-role are not part of the created nodes
resource_delete:
	$(INFRA_CMD) ${PROVIDER} resource delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f ${PROMBENCH_DIR}/${BENCHMARK_DIRECTORY}/benchmark/1c_cluster-role-binding.yaml \
		-f ${PROMBENCH_DIR}/${BENCHMARK_DIRECTORY}/benchmark/1a_namespace.yaml

node_delete:
	$(INFRA_CMD) ${PROVIDER} nodes delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f ${PROMBENCH_DIR}/${BENCHMARK_DIRECTORY}/nodes_${PROVIDER}.yaml

all_nodes_running:
	$(INFRA_CMD) ${PROVIDER} nodes check-running -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} -v SEPARATOR:${SEPARATOR} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f ${PROMBENCH_DIR}/${BENCHMARK_DIRECTORY}/nodes_${PROVIDER}.yaml

all_nodes_deleted:
	$(INFRA_CMD) ${PROVIDER} nodes check-deleted -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} -v SEPARATOR:${SEPARATOR} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f ${PROMBENCH_DIR}/${BENCHMARK_DIRECTORY}/nodes_${PROVIDER}.yaml
