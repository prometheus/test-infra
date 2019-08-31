#!/bin/sh
set -e

# Delete /tmp/READY to mark the container as un-ready, can be used in readiness probe
rm /tmp/READY || true

# Wait until all_nodepools_running returns a success code then proceed
until make all_nodepools_running; do
  echo "waiting for nodepools to be created";
  sleep 10;
done;

# Stop deploy if running
supervisorctl stop deploy

# Start cleaningup benchmarking components
make clean
# FIXME : what happens when there's an error in make clean