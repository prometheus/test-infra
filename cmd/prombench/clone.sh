#!/bin/sh

# clone the prombench repo for latest Makefile and manifest files
git clone $PROMBENCH_REPO $PROMBENCH_DIR

# copy the prombench binary to the cloned directory
cp /usr/bin/prombench $PROMBENCH_DIR/

# execute arguments passed to the image
$@
