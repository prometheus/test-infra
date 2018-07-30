#!/bin/bash

PR_NUMBER=$1

if [ -z "$PR_NUMBER" ]; then echo "ERROR::PR NUMBER is missing in argument" && exit 1; fi

DIR="/go/src/github.com/prometheus/prometheus"

printf "\n\n>> Cloning repository 'prometheus/prometheus' \n\n"

if ! git clone https://github.com/prometheus/prometheus.git $DIR; then printf "ERROR:: Cloning of repo 'prometheus/prometheus' failed" && exit 1; fi

cd $DIR || exit 1

printf "\n\n>> Fetching Pull Request 'https://github.com/prometheus/prometheus/pull/%s' \n\n" "$PR_NUMBER"

if ! git fetch origin pull/"$PR_NUMBER"/head:pr-branch; then printf "ERROR:: Fetching of PR %s failed" "$PR_NUMBER" && exit 1; fi

git checkout pr-branch
printf "\n\n>> Creating prometheus binaries\n\n"

if ! make build; then printf "ERROR:: Building of binaries failed" && exit 1; fi

printf "\n\n>> Starting prometheus\n\n"
./prometheus --config.file=/etc/prometheus/config/prometheus.yaml \
             --storage.tsdb.path=/data \
             --web.console.libraries=${DIR}/console_libraries \
             --web.console.templates=${DIR}/consoles