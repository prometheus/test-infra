#!/bin/bash

PR_NUMBER=$1

if [ -z "$PR_NUMBER" ]
then
	echo "ERROR:: PR NUMBER is missing in argument"
	exit 1
fi

DIR="/go/src/github.com/prometheus/prometheus"

printf "\n\n>> Fetching Pull Request\n\n"
git clone https://github.com/prometheus/prometheus.git $DIR

cd $DIR
git fetch origin pull/$PR_NUMBER/head:pr-branch
git checkout pr-branch
printf "\n\n>> Creating prometheus binaries\n\n"
make build
printf "\n\n>> Starting prometheus\n\n"
./prometheus --config.file=/etc/prometheus/config/prometheus.yaml \
             --storage.tsdb.path=/data \
             --web.console.libraries=${DIR}/console_libraries \
             --web.console.templates=${DIR}/consoles