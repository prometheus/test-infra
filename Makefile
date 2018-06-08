# Copyright 2016 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

all: build test

BENCHMARK_VERSION := 1.0
GO           ?= go
GOFMT        ?= $(GO)fmt
FIRST_GOPATH := $(firstword $(subst :, ,$(shell $(GO) env GOPATH)))
STATICCHECK  := $(FIRST_GOPATH)/bin/staticcheck

# Build and push specific variables.
REGISTRY ?= gcr.io
PROJECT  ?= prometheus-test-204522

prometheus_style:
	@echo ">> checking code style"
	! $(GOFMT) -d $$(find . -path ./vendor -prune -o -name '*.go' -print) | grep '^'

staticcheck: $(STATICCHECK)
	@echo ">> running staticcheck"
	$(STATICCHECK) -ignore "$(STATICCHECK_IGNORE)" $(pkgs)

unused: $(GOVENDOR)
	@echo ">> running check for unused packages"
	@$(GOVENDOR) list +unused | grep . && exit 1 || echo 'No unused packages'

build:
	@echo ">> building binaries"
	go install ./... -o benchmark

test: prometheus_style staticcheck unused build

push:
	docker build -t "$(REGISTRY)/$(PROJECT)/prombench:$(BENCHMARK_VERSION)" .
	docker push "$(REGISTRY)/$(PROJECT)/prombench:$(BENCHMARK_VERSION)"

.PHONY: build push test prometheus_style staticcheck unused