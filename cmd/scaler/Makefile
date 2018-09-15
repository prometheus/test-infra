PROMBENCH_CMD        = ./scaler
DOCKER_TAG = docker.io/prombench/scaler:2.0.0

scale:
	$(PROMBENCH_CMD) scale -v PR_NUMBER:${PR_NUMBER} ${SCALE_DOWN_REPLICAS} ${SCALE_UP_REPLICAS} ${INTERVAL} \
		-f components/prombench/manifests/benchmark/2_fake-webserver.yaml

build:
	@go version | grep go1.11 || exit  "Requires golang 1.11 with support for modules!"
	@GO111MODULE=on GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build scaler.go

clean:
	@rm scaler

docker: build
	# Copying fake-webserver.yaml here because current docker does not support building out of the current context
	@mkdir -p tmp
	@cp ../../components/prombench/manifests/benchmark/2_fake-webserver.yaml tmp/2_fake-webserver.yaml
	@docker build -t $(DOCKER_TAG) .
	@docker push $(DOCKER_TAG)
	@rm -rf tmp

.PHONY: scale build clean docker