PROMBENCH_CMD        = ./scaler
DOCKER_TAG = docker.io/prombench/scaler:2.0.0

scale:
	$(PROMBENCH_CMD) scale -v PR_NUMBER:${PR_NUMBER} ${SCALE_DOWN_REPLICAS} ${SCALE_UP_REPLICAS} ${INTERVAL} \
		-f components/prombench/manifests/benchmark/2_fake-webserver.yaml

build:
	@go version | grep go1.11 || exit  "Requires golang 1.11 with support for modules!"
	@GO111MODULE=on go build scaler.go

clean:
	@rm scaler

docker: build
	@docker build -t $(DOCKER_TAG) .
	@docker push $(DOCKER_TAG)

.PHONY: scale build clean docker