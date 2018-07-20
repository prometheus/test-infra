PROMBENCH_CMD        = /bin/prombench
DOCKER_TAG = docker.io/sipian/prombench:v2.0.0

deploy:
	$(PROMBENCH_CMD) gke nodepool create -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f config/prombench.yaml

	$(PROMBENCH_CMD) gke resource apply -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-v PROMETHEUS_1_NAME:${PROMETHEUS_1_NAME} -v PROMETHEUS_1_IMAGE:${PROMETHEUS_1_IMAGE} \
		-v PROMETHEUS_2_NAME:${PROMETHEUS_2_NAME} -v PROMETHEUS_2_IMAGE:${PROMETHEUS_2_IMAGE} \
		-f manifests

clean:
	$(PROMBENCH_CMD) gke resource delete -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-v PROMETHEUS_1_NAME:${PROMETHEUS_1_NAME} -v PROMETHEUS_1_IMAGE:${PROMETHEUS_1_IMAGE} \
		-v PROMETHEUS_2_NAME:${PROMETHEUS_2_NAME} -v PROMETHEUS_2_IMAGE:${PROMETHEUS_2_IMAGE} \
		-f manifests/1a_namespace.yaml -f manifests/1c_cluster-role-binding.yaml

	$(PROMBENCH_CMD) gke nodepool delete -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f config/prombench.yaml

build:
	@vgo build -o prombench cmd/prombench/main.go

docker: build
	@docker build -t $(DOCKER_TAG) .
	@docker push $(DOCKER_TAG)

.PHONY: deploy clean build docker