include Makefile.common

PROMBENCH_CMD        = ./prombench

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

.PHONY: docker-prombench-build docker-prombench-tag-latest docker-prombench-publish
docker-prombench-build: DOCKER_IMAGE_NAME=prombench
docker-prombench-build: DOCKERFILE_PATH=./
docker-prombench-build: docker
docker-prombench-tag-latest: DOCKER_IMAGE_NAME=prombench
docker-prombench-tag-latest: docker-tag-latest
docker-prombench-publish: DOCKER_IMAGE_NAME=prombench
docker-prombench-publish: docker-publish

.PHONY: docker-fake-webserver-build docker-fake-webserver-tag-latest docker-fake-webserver-publish
docker-fake-webserver-build: DOCKER_IMAGE_NAME=fake-webserver
docker-fake-webserver-build: DOCKERFILE_PATH=./cmd/fake-webserver/
docker-fake-webserver-build: docker
docker-fake-webserver-tag-latest: DOCKER_IMAGE_NAME=fake-webserver
docker-fake-webserver-tag-latest: docker-tag-latest
docker-fake-webserver-publish: DOCKER_IMAGE_NAME=fake-webserver
docker-fake-webserver-publish: docker-publish

.PHONY: docker-scaler-build docker-scaler-tag-latest docker-scaler-publish
docker-scaler-build: DOCKER_IMAGE_NAME=scaler
docker-scaler-build: DOCKERFILE_PATH=./cmd/scaler/
docker-scaler-build: docker
docker-scaler-tag-latest: DOCKER_IMAGE_NAME=scaler
docker-scaler-tag-latest: docker-tag-latest
docker-scaler-publish: DOCKER_IMAGE_NAME=scaler
docker-scaler-publish: docker-publish

.PHONY: docker-prometheus-builder-build docker-prometheus-builder-tag-latest docker-prometheus-builder-publish
docker-prometheus-builder-build: DOCKER_IMAGE_NAME=scaler
docker-prometheus-builder-build: DOCKERFILE_PATH=./cmd/scaler/
docker-prometheus-builder-build: docker
docker-prometheus-builder-tag-latest: DOCKER_IMAGE_NAME=scaler
docker-prometheus-builder-tag-latest: docker-tag-latest
docker-prometheus-builder-publish: DOCKER_IMAGE_NAME=scaler
docker-prometheus-builder-publish: docker-publish
