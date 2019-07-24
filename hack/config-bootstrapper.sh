#!/bin/bash
set -euo pipefail

CB_DOCKER_IMAGE="gcr.io/k8s-prow/config-bootstrapper:v20190608-493ef838c"
dir=$(realpath "$(dirname "${BASH_SOURCE}")/..")

docker run --rm \
    -v $dir:/prombench \
    -v $HOME/.kube:/kube \
    --network host \
    $CB_DOCKER_IMAGE \
    --dry-run=false \
    --source-path /prombench \
    --config-path /prombench/config/prow/config.yaml \
    --plugin-config /prombench/config/prow/plugins.yaml \
    --kubeconfig=/kube/$(basename $KUBECONFIG)
