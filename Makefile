DOCKER_REPO             ?= prombench

.PHONY: all
all: precheck style check_license lint build test unused

include Makefile.common
