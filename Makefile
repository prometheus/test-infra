PROMBENCH_CMD        = /bin/prombench
DOCKER_TAG = gcr.io/prometheus-test-204522/prombench:v0.1.0

deploy:
	$(PROMBENCH_CMD) gke cluster scaleUp -a /etc/serviceaccount/service-account.json -c config/node-pool.yaml \
		-v ZONE:${ZONE} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER}

	$(PROMBENCH_CMD) gke resource apply -a /etc/serviceaccount/service-account.json -c config/node-pool.yaml -f manifests \
		-v ZONE:${ZONE} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-v PROMETHEUS_1_NAME:${PROMETHEUS_1_NAME} -v PROMETHEUS_1_IMAGE:${PROMETHEUS_1_IMAGE} \
		-v PROMETHEUS_2_NAME:${PROMETHEUS_2_NAME} -v PROMETHEUS_2_IMAGE:${PROMETHEUS_2_IMAGE}

clean:
	$(PROMBENCH_CMD) gke resource delete -a /etc/serviceaccount/service-account.json -c config/node-pool.yaml  -f manifests \
		-v ZONE:${ZONE} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-v PROMETHEUS_1_NAME:${PROMETHEUS_1_NAME} -v PROMETHEUS_1_IMAGE:${PROMETHEUS_1_IMAGE} \
		-v PROMETHEUS_2_NAME:${PROMETHEUS_2_NAME} -v PROMETHEUS_2_IMAGE:${PROMETHEUS_2_IMAGE}

	$(PROMBENCH_CMD) gke cluster scaleDown -a /etc/serviceaccount/service-account.json -c config/node-pool.yaml \
		-v ZONE:${ZONE} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER}

build:
	go build -o prombench cmd/prombench/main.go

docker:
	@docker build -t $(DOCKER_TAG) .
	@docker push $(DOCKER_TAG)

.PHONY: deploy clean build docker
