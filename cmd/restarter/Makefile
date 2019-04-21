DOCKER_TAG = docker.io/geekodour/restarter:1.0.0

build:
	@go version | grep go1.12 || exit  "Requires golang 1.12 with support for modules!"
	@GO111MODULE=on GOARCH=amd64 CGO_ENABLED=0 GOOS=linux go build restarter.go

clean:
	@rm restarter

docker: build
	@docker build -t $(DOCKER_TAG) .
	@docker push $(DOCKER_TAG)

.PHONY: build clean docker
