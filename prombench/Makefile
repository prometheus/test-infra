INFRA_CMD        ?= ../infra/infra

.PHONY: deploy clean
deploy: nodepool_create resource_apply
# GCP sometimes takes longer than 30 tries when trying to delete nodepools
# if k8s resources are not already cleared
clean: resource_delete nodepool_delete

nodepool_create:
	$(INFRA_CMD) gke nodes create -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodes_gke.yaml

resource_apply:
	$(INFRA_CMD) gke resource apply -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} -v DOMAIN_NAME:${DOMAIN_NAME} \
		-v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
		-f manifests/prombench/benchmark

# Required because namespace and cluster-role are not part of the created nodepools
resource_delete:
	$(INFRA_CMD) gke resource delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/benchmark/1c_cluster-role-binding.yaml \
		-f manifests/prombench/benchmark/1a_namespace.yaml

nodepool_delete:
	$(INFRA_CMD) gke nodes delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodes_gke.yaml

all_nodepools_running:
	$(INFRA_CMD) gke nodes check-running -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodes_gke.yaml

all_nodepools_deleted:
	$(INFRA_CMD) gke nodes check-deleted -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodes_gke.yaml
