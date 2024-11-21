#!/bin/bash

DIR="/go/src/github.com/prometheus/prometheus"

if [[ -z $PR_NUMBER || -z $VOLUME_DIR || -z $STORAGE || -z $GITHUB_ORG || -z $GITHUB_REPO ]]; then
    echo "ERROR:: environment variables not set correctly"
    exit 1;
fi

# Clone the repository with a shallow clone
echo ">> Cloning repository $GITHUB_ORG/$GITHUB_REPO (shallow clone)"
if ! git clone --depth 1 https://github.com/$GITHUB_ORG/$GITHUB_REPO.git $DIR; then
    echo "ERROR:: Cloning of repo $GITHUB_ORG/$GITHUB_REPO failed"
    exit 1;
fi

cd $DIR || exit 1

echo ">> Fetching Pull Request $GITHUB_ORG/$GITHUB_REPO/pull/$PR_NUMBER"
if ! git fetch origin pull/$PR_NUMBER/head:pr-branch; then
    echo "ERROR:: Fetching of PR $PR_NUMBER failed"
    exit 1;
fi

git checkout pr-branch

# Here, STORAGE is specified as an  environment variable of the prometheus-builder init container, 
# where it will copy the key.yml file from the Prometheus directory to the volume section of the
# emptyDir. This file will later be used by the data-downloader init container.
if [ -f "$DIR/key.yml" ]; then
    echo "INFO:: key.yml file is Present on $DIR/key.yml directory so download the block from ObjecStorage."
    cp  "$DIR/key.yml" "$STORAGE/key.yml"
else
    echo "INFO:: key.yml File does not exist on $DIR/key.yml directory so data is not downloaded from ObjectStorage."
fi

echo ">> Creating prometheus binaries"
if ! make build PROMU_BINARIES="prometheus"; then
    echo "ERROR:: Building of binaries failed"
    exit 1;
fi

echo ">> Copy files to volume"
cp prometheus               $VOLUME_DIR/prometheus
