#!/bin/sh

# clone the test-infra repo for latest Makefile and manifest files
git clone $TEST_INFRA_REPO $TEST_INFRA_DIR

# copy the prombench binary to the cloned directory
cp /usr/bin/prombench $PROMBENCH_DIR/
cp -r $TEST_INFRA_DIR/prombench/* $PROMBENCH_DIR/ 