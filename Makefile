PROMBENCH_CMD        = ./prombench
DOCKER_TAG = docker.io/prombench/prombench:2.0.0

deploy:
	$(PROMBENCH_CMD) gke nodepool create -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f  components/prombench/nodepools.yaml

	$(PROMBENCH_CMD) gke resource apply -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} \
		-v PR_NUMBER:${PR_NUMBER} -v RELEASE:${RELEASE} \
		-f components/prombench/manifests/benchmark

clean:
	$(PROMBENCH_CMD) gke resource delete -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f components/prombench/manifests/benchmark/1a_namespace.yaml -f components/prombench/manifests/benchmark/1c_cluster-role-binding.yaml

	$(PROMBENCH_CMD) gke nodepool delete -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f components/prombench/nodepools.yaml

build:
	@vgo build -o prombench cmd/prombench/main.go

docker: build
	@docker build -t $(DOCKER_TAG) .
	@docker push $(DOCKER_TAG)

.PHONY: deploy clean build docker