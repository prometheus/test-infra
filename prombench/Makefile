INFRA_CMD        ?= ../infra/infra

PROVIDER 		 ?= gke

ifeq ($(PROVIDER),)
PROVIDER = gke
endif

.PHONY: deploy clean
deploy: node_create resource_apply
# GCP sometimes takes longer than 30 tries when trying to delete nodes
# if k8s resources are not already cleared
clean: resource_delete node_delete

nodepool_create:
	${INFRA_CMD} ${PROVIDER} nodes create -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} -v SEPARATOR:${SEPARATOR} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \ 
		-f manifests/prombench/nodes_${PROVIDER}.yaml

resource_apply:
	$(INFRA_CMD) ${PROVIDER} resource apply -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v REGION:${REGION} \ 
		-v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} -v DOMAIN_NAME:${DOMAIN_NAME} \
		-v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
		-f manifests/prombench/benchmark

# Required because namespace and cluster-role are not part of the created nodes
resource_delete:
	$(INFRA_CMD) ${PROVIDER} resource delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v REGION:${REGION} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/benchmark/1c_cluster-role-binding.yaml \
		-f manifests/prombench/benchmark/1a_namespace.yaml

nodepool_delete:
	$(INFRA_CMD) ${PROVIDER} nodes delete -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} -v SEPARATOR:${SEPARATOR} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodes_${PROVIDER}.yaml

all_nodepools_running:
	$(INFRA_CMD) ${PROVIDER} nodes check-running -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} -v SEPARATOR:${SEPARATOR} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodes_${PROVIDER}.yaml	

all_nodepools_deleted:
	$(INFRA_CMD) ${PROVIDER} nodes check-deleted -a ${AUTH_FILE} \
		-v ZONE:${ZONE} -v GKE_PROJECT_ID:${GKE_PROJECT_ID} \
		-v EKS_WORKER_ROLE_ARN:${EKS_WORKER_ROLE_ARN} -v EKS_CLUSTER_ROLE_ARN:${EKS_CLUSTER_ROLE_ARN} \
		-v EKS_SUBNET_IDS:${EKS_SUBNET_IDS} -v SEPARATOR:${SEPARATOR} \
		-v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f manifests/prombench/nodes_${PROVIDER}.yaml
