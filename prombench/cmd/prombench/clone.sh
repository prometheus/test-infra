#!/bin/sh

# clone the test-infra repo for latest Makefile and manifest files
git clone $TEST_INFRA_REPO $TEST_INFRA_DIR

# copy the prombench binary to the cloned directory
cp /usr/bin/prombench $PROMBENCH_DIR/

# execute arguments passed to the image
# eval is needed so that bash keywords are not run as commands
eval "$@"
