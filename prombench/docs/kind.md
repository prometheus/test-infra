# Prombench in KIND

Run Prombench tests in [Kubernetes In Docker](https://kind.sigs.k8s.io/).

## Table of Contents

1. [Setup Prombench](#setup-prombench)
   - [Install KIND](#install-kind)
   - [Create the KIND Cluster](#create-the-kind-cluster)
   - [Deploy Monitoring Components](#deploy-monitoring-components)
2. [Usage](#usage)
   - [Start a Benchmarking Test Manually](#start-a-benchmarking-test-manually)
   - [Deleting Benchmark Infrastructure](#deleting-benchmark-infrastructure)

---

## Setup Prombench

### 1. Install KIND

Follow the [KIND installation guide](https://kind.sigs.k8s.io/docs/user/quick-start/) to install KIND on your system.

### 2. Create the KIND Cluster

#### a. Build the Infra CLI Tool

1. Navigate to the `infra/` directory:
   ```bash
   cd infra/
   ```
2. Build the Infra CLI tool:
   ```bash
   go build .
   ```
3. Navigate to the `prombench/` directory:
   ```bash
   cd ../prombench/
   ```

#### b. Create a Multi-Node KIND Cluster

1. Set the necessary environment variables:
   ```bash
   export CLUSTER_NAME=prombench
   export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
   ```
2. Deploy the cluster:
   ```bash
   ../infra/infra kind cluster create -v PR_NUMBER:$PR_NUMBER -v CLUSTER_NAME:$CLUSTER_NAME \
       -f manifests/cluster_kind.yaml
   ```
3. Remove the taint from the `prombench-control-plane` node for deploying the nginx-ingress-controller:
   ```bash
   kubectl --context kind-$CLUSTER_NAME taint nodes $CLUSTER_NAME-control-plane node-role.kubernetes.io/control-plane-
   ```

### 3. Deploy Monitoring Components

> Collecting, monitoring, and displaying the test results and logs.

#### a. [Optional] Generate a GitHub Auth Token

If used with the GitHub integration:

1. Login with the [Prombot account](https://github.com/prombot) and generate a [new auth token](https://github.com/settings/tokens) with the following permissions:
   - `public_repo`
   - `read:org`
   - `write:discussion`

2. Set the following environment variables:
   ```bash
   export GRAFANA_ADMIN_PASSWORD=password
   export DOMAIN_NAME=prombench.prometheus.io # Can be set to any other custom domain or an empty string when not used with the Github integration.
   export OAUTH_TOKEN=<generated token from github or set to an empty string " ">
   export WH_SECRET=<github webhook secret>
   export GITHUB_ORG=prometheus
   export GITHUB_REPO=prometheus
   export SERVICEACCOUNT_CLIENT_EMAIL=<Your Email address>
   ```

3. Deploy the [nginx-ingress-controller](https://github.com/kubernetes/ingress-nginx), Prometheus-Meta, Loki, Grafana, Alertmanager & Github Notifier:
   ```bash
   ../infra/infra kind resource apply -v CLUSTER_NAME:$CLUSTER_NAME -v DOMAIN_NAME:$DOMAIN_NAME \
       -v GRAFANA_ADMIN_PASSWORD:$GRAFANA_ADMIN_PASSWORD \
       -v OAUTH_TOKEN="$(printf $OAUTH_TOKEN | base64 -w 0)" \
       -v WH_SECRET="$(printf $WH_SECRET | base64 -w 0)" \
       -v GITHUB_ORG:$GITHUB_ORG -v GITHUB_REPO:$GITHUB_REPO \
       -v SERVICEACCOUNT_CLIENT_EMAIL:$SERVICEACCOUNT_CLIENT_EMAIL \
       -f manifests/cluster-infra
   ```

4. Set the `NODE_NAME`, `INTERNAL_IP`, and `NODE_PORT` environment variables:
   ```bash
   export NODE_NAME=$(kubectl --context kind-$CLUSTER_NAME get pod -l "app=grafana" -o=jsonpath='{.items[*].spec.nodeName}')
   export INTERNAL_IP=$(kubectl --context kind-$CLUSTER_NAME get nodes $NODE_NAME -o jsonpath='{.status.addresses[?(@.type=="InternalIP")].address}')
   export NODE_PORT=$(kubectl --context kind-$CLUSTER_NAME get -o jsonpath="{.spec.ports[0].nodePort}" services grafana)
   ```

5. The services will be accessible at:
   ```bash
   echo "Grafana: http://$INTERNAL_IP:$NODE_PORT/grafana"
   echo "Prometheus: http://$INTERNAL_IP:$NODE_PORT/prometheus-meta"
   echo "Logs: http://$INTERNAL_IP:$NODE_PORT/grafana/explore"
   echo "Profiles: http://$INTERNAL_IP:$NODE_PORT/profiles"
   ```

## Usage

### 1. Start a Benchmarking Test Manually

1. Set the following environment variables:
   ```bash
   export RELEASE=<master/main or any prometheus release(ex: v2.3.0) >
   export PR_NUMBER=<PR to benchmark against the selected $RELEASE>
   ```

2. Deploy the Kubernetes objects:
   > **_Note:_** If you encounter a `too many files open` error caused by promtail, increase the default value of `/proc/sys/fs/inotify/max_user_instances` from 128 to 512:
   > ```bash
   > sudo sysctl fs.inotify.max_user_instances=512
   > ```
   > **_Tip:_** When using prombench locally, it is recommended to build all the Docker images of tools under the `tools/` directory. Instructions are available in their respective `README.md` files.
   
   ```bash
   ../infra/infra kind resource apply -v CLUSTER_NAME:$CLUSTER_NAME \
       -v PR_NUMBER:$PR_NUMBER -v RELEASE:$RELEASE -v DOMAIN_NAME:$DOMAIN_NAME \
       -v GITHUB_ORG:${GITHUB_ORG} -v GITHUB_REPO:${GITHUB_REPO} \
       -f manifests/prombench/benchmark
   ```

### 2. Deleting Benchmark Infrastructure

1. To delete the benchmark infrastructure, run:
   ```bash
   ../infra/infra kind cluster delete -v PR_NUMBER:$PR_NUMBER -v CLUSTER_NAME:$CLUSTER_NAME -f manifests/cluster_kind.yaml
   ```
