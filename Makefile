PROMBENCH_CMD        = ./prombench
DOCKER_TAG = docker.io/sipian/prombench:v2.0.0

#Prow config has the following args set in it's configuration during deployment 
#	PROJECT_ID
#	ZONE
#	CLUSTER_NAME
#
#When the start-benchmark prow-job is created, the benchmark plugin adds the following args
#	ACTION - [release|pr|clean]
#	PR_NUMBER
#	PROMETHEUS_1_NAME
#	PROMETHEUS_1_IMAGE
#	PROMETHEUS_2_NAME
#	PROMETHEUS_2_IMAGE
#The values of these args are not constant are are dependent on /benchmark pr|release

#For /benchmark release
# 	PROMETHEUS_1_NAME is master
#	PROMETHEUS_1_IMAGE is https://quay.io/prometheus/prometheus:master
#
# 	PROMETHEUS_2_NAME is [release-number|latest]
# 	PROMETHEUS_2_IMAGE is https://quay.io/prometheus/prometheus:[v<RELEASE_NUMBER>|latest]

#For /benchmark pr
# 	PROMETHEUS_1_NAME is master
#	PROMETHEUS_1_IMAGE is https://quay.io/prometheus/prometheus:master
#
# 	PROMETHEUS_2_NAME is pr-<PR_NUMBER>
# 	PROMETHEUS_2_IMAGE is $DOCKER_TAG (with args as make start-pr)


create-nodepool:
	printf ">> Creating NodePools"
	$(PROMBENCH_CMD) gke nodepool create -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f  components/prombench/nodepools.yaml

DIR := "/go/src/github.com/${REPO_OWNER}/${REPO_NAME}"
start-pr:
	mkdir -p /go/src/github.com
	printf "\n\n>> Fetching Pull Request"
	git clone https://github.com/${REPO_OWNER}/${REPO_NAME} ${DIR}

	cd ${DIR} && \
	git fetch origin pull/${PR_NUMBER}/head:pr-branch && \
	git checkout pr-branch && \
	printf "\n\n>> Creating prometheus binaries" && \
	make build && \
	printf "\n\n>> Starting prometheus" && \
	./prometheus --config.file=/etc/prometheus/config/prometheus.yaml \
          		 --storage.tsdb.path=/data \
				 --web.console.libraries=${DIR}/console_libraries \
            	 --web.console.templates=${DIR}/consoles

deploy-pr: create-nodepool
	printf ">> Deploying Prombench components for PR"
	$(PROMBENCH_CMD) gke resource apply -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-v PROMETHEUS_1_NAME:${PROMETHEUS_1_NAME} -v PROMETHEUS_1_IMAGE:${PROMETHEUS_1_IMAGE} \
		-v PROMETHEUS_2_NAME:${PROMETHEUS_2_NAME} -v PROMETHEUS_2_IMAGE:${DOCKER_TAG} \
		-v REPO_OWNER:${REPO_OWNER} -v REPO_NAME:${REPO_NAME} \
		-f components/prombench/manifests/benchmark/namespace.yaml \
		-f components/prombench/manifests/benchmark/serviceaccount.yaml \
		-f components/prombench/manifests/benchmark/cluster-role-binding.yaml \
		-f components/prombench/manifests/benchmark/fake-webserver.yaml \
		-f components/prombench/manifests/benchmark/loadgen.yaml \
		-f components/prombench/manifests/benchmark/prometheus-pr.yaml \
		-f components/prombench/manifests/benchmark/node-exporter.yaml 	#node-exporter should be deployed after prometheus(to use pod-affinity)

deploy-release: create-nodepool
	printf ">> Deploying Prombench components for release"
	$(PROMBENCH_CMD) gke resource apply -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-v PROMETHEUS_1_NAME:${PROMETHEUS_1_NAME} -v PROMETHEUS_1_IMAGE:${PROMETHEUS_1_IMAGE} \
		-v PROMETHEUS_2_NAME:${PROMETHEUS_2_NAME} -v PROMETHEUS_2_IMAGE:${PROMETHEUS_2_IMAGE} \
		-f components/prombench/manifests/benchmark/namespace.yaml \
		-f components/prombench/manifests/benchmark/serviceaccount.yaml \
		-f components/prombench/manifests/benchmark/cluster-role-binding.yaml \
		-f components/prombench/manifests/benchmark/fake-webserver.yaml \
		-f components/prombench/manifests/benchmark/loadgen.yaml \
		-f components/prombench/manifests/benchmark/prometheus-release.yaml \
		-f components/prombench/manifests/benchmark/node-exporter.yaml 	#node-exporter should be deployed after prometheus(to use pod-affinity)

deploy:
ifeq ($(ACTION),release)
deploy:deploy-release
else ifeq ($(ACTION),pr)
deploy:deploy-pr
endif

clean:
	printf ">> Cleaning Prombench components"
	$(PROMBENCH_CMD) gke resource delete -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-v PROMETHEUS_1_NAME:${PROMETHEUS_1_NAME} -v PROMETHEUS_1_IMAGE:${PROMETHEUS_1_IMAGE} \
		-v PROMETHEUS_2_NAME:${PROMETHEUS_2_NAME} -v PROMETHEUS_2_IMAGE:${PROMETHEUS_2_IMAGE} \
		-f components/prombench/manifests/benchmark/namespace.yaml \
		-f components/prombench/manifests/benchmark/cluster-role-binding.yaml

	printf ">> Cleaning NodePools components"
	$(PROMBENCH_CMD) gke nodepool delete -a /etc/serviceaccount/service-account.json \
		-v ZONE:${ZONE} -v PROJECT_ID:${PROJECT_ID} -v CLUSTER_NAME:${CLUSTER_NAME} -v PR_NUMBER:${PR_NUMBER} \
		-f components/prombench/nodepools.yaml

build:
	vgo build -o prombench cmd/prombench/main.go

docker: build
	docker build -t $(DOCKER_TAG) .
	#docker push $(DOCKER_TAG)

.PHONY: create-nodepool start-pr deploy-pr deploy-release clean build docker