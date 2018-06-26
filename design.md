# Prombench

The proposal for Prometheus CI project can be found [here](https://docs.google.com/document/d/1aCGHS0hOrh3LiQLuOa1EWA6knF7HmqWbhp3ev66hB7Y/edit?ouid=118160464041419930165&usp=docs_home&ths=true)

The TODO list can be found [here](https://github.com/sipian/prombench/issues/5)


- Assume [Prow](https://github.com/sipian/test-infra/tree/prometheus-prow/prow/) is already running on a GKE cluster.
Instructions on how to deploy prow-cluster can be found [here](prow-files/deploy-prow)
- When Prombench is triggered using `/benchmark pr` or `/benchmark release`, the following environment variables are set in the Prowjob [start-benchmark](https://github.com/sipian/test-infra/blob/prometheus-prow/prow/config-prometheus.yaml#L62):
	- ZONE : zone of the prow cluster
	- CLUSTER_NAME : Name of the prow cluster
	- PR_NUMBER : Number of the PR where this comment was written
	- PROMETHEUS_1_NAME
	- PROMETHEUS_1_IMAGE
	- PROMETHEUS_2_NAME
	- PROMETHEUS_2_IMAGE

- 2 new [nodepools](config/cluster.yaml) are created in the prow cluster :: `prometheus-<PR_NUMBER>` and `nodes-<PR_NUMBER>`
- [Prombench](manifests) is deployed on these nodepools in a new namespace `prombench-<PR_NUMBER>` (Only one Prombench instance can run on a PR)

- When `/benchmark delete` is triggered, the nodepool and namespace is deleted