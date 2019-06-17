## Overview of the manifest files
The `/manifest` directory contains all the kubernetes manifest files.
- `cluster.yaml` : This is used to create the GKE cluster.
- `cluster-infra/` : These are the persistent cluster infrastructure resources.
- `prombench/` : These resources are created and destoryed for each prombench test.
- `prow/` : Resources for deploying [prow](https://github.com/kubernetes/test-infra/tree/master/prow/), which is used to trigger tests from GitHub comments.
