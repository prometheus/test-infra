# Prombench

The proposal for Prometheus CI project can be found [here](https://docs.google.com/document/d/1aCGHS0hOrh3LiQLuOa1EWA6knF7HmqWbhp3ev66hB7Y/edit?ouid=118160464041419930165&usp=docs_home&ths=true)

The TODO list can be found [here](https://github.com/sipian/prombench/issues/5)


- Assume [Prow](https://github.com/sipian/test-infra/tree/prometheus-prow/prow/) is already running on a GKE cluster. <br/> Instructions on how to deploy a prow-cluster can be found [here](prow-files/deploy-prow).<br/>The benchmark plugin added to prow can be found [here](https://github.com/sipian/test-infra/tree/prometheus-prow/prow/plugins/benchmark)

- When Prombench is triggered using `/benchmark pr` or `/benchmark release [version_number(ex:2.3.0-rc.1)|Default:latest]`, the following environment variables are set in the Prowjob [start-benchmark](https://github.com/sipian/test-infra/blob/prometheus-prow/prow/config-prometheus.yaml#L34):
	- ZONE : zone of the prow cluster
	- CLUSTER_NAME : Name of the prow cluster
	- PR_NUMBER : Number of the PR where this comment was written
	- PROMETHEUS_1_NAME
	- PROMETHEUS_1_IMAGE
	- PROMETHEUS_2_NAME
	- PROMETHEUS_2_IMAGE

- In case of `/benchmark pr`, initially a docker image is made for the PR. 

- 2 new [nodepools](config/node-pool.yaml) are created in the prow cluster :: `prometheus-<PR_NUMBER>` and `nodes-<PR_NUMBER>`

- [Prombench](manifests) is deployed on these nodepools in a new namespace `prombench-<PR_NUMBER>` (Only one Prombench instance can run on a PR)

- When `/benchmark cancel` is triggered, the nodepools and namespace are deleted
