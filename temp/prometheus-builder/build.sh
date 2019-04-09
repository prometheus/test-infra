#!/bin/bash

DIR="/go/src/github.com/prometheus/prometheus"

if [ -z "$PR_NUMBER" ]; then
    echo "ERROR::PR NUMBER is missing in argument"
    exit 1;
fi

echo ">> Cloning repository prometheus/prometheus"
if ! git clone https://github.com/prometheus/prometheus.git $DIR; then
    echo "ERROR:: Cloning of repo prometheus/prometheus failed"
    exit 1;
fi

cd $DIR || exit 1

echo ">> Fetching Pull Request prometheus/prometheus/pull/$PR_NUMBER"
if ! git fetch origin pull/$PR_NUMBER/head:pr-branch; then
    echo "ERROR:: Fetching of PR $PR_NUMBER failed"
    exit 1;
fi

git checkout pr-branch

echo ">> Creating prometheus binaries"
if ! make build; then
    echo "ERROR:: Building of binaries failed"
    exit 1;
fi

echo ">> Copy files to volume"
cp prometheus               $VOLUME_DIR/prometheus
cp -r console_libraries/    $VOLUME_DIR
cp -r consoles/             $VOLUME_DIR