#!/bin/sh
set -e

# Wait until all_nodepools_deleted returns a success code then proceed
until make all_nodepools_deleted; do
  echo "waiting for nodepools to be deleted";
  sleep 10;
done;

# Start deploying benchmarking components
make deploy

# Create /tmp/READY to mark the container as ready, can be used in readiness probe
touch /tmp/READY

# Log success and sleep
echo 'deploy succeeded, now sleeping'
sleep 99999999