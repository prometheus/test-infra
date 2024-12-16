# Automated Prometheus E2E Testing and Benchmarking

![Prombench Design](design.svg)

This setup leverages **GitHub Actions** and **Google Kubernetes Engine (GKE)**, but is designed to be extendable to other Kubernetes providers.

## Configuration Files

The `./manifest` directory contains configuration files. We can outline :

- **`./manifest/cluster_gke.yaml`**: Creates the Main Node in GKE.
- **`./manifest/cluster_eks.yaml`**: Creates the Main Node in EKS.
- **`./manifest/cluster-infra/`**: Contains persistent components of the Main Node.
- **`./manifest/prombench/`**: Resources created and destroyed for each Prombench test. See [`its README.md`](./manifests/prombench/README.md) for details.

## Prombench Setup

Prombench can be run on different providers. Follow these instructions based on your provider:

- [Google Kubernetes Engine (GKE)](docs/gke.md)
- [Kubernetes In Docker (KIND)](docs/kind.md)
- [Elastic Kubernetes Service (EKS)](docs/eks.md)

### Setting Up GitHub Actions

1. Place a workflow file in the `.github` directory of your repository. Refer to the [Prometheus GitHub repository](https://github.com/prometheus/prometheus) for an example.

2. Create a GitHub Action secret `TEST_INFRA_PROVIDER_AUTH` with the base64 encoded content of the `AUTH_FILE`:

   ```bash
   cat $AUTH_FILE | base64 -w 0
   ```
    
3. Configure webhook to cluster's comment-monitor as described [here](../tools/comment-monitor/README.md#setting-up-the-github-webhook).

## Prombench Usage

### Triggering Tests via GitHub Comment

**Starting Tests:**

- `/prombench main` or `/prombench master` - Compare PR with the main/master branch.
- `/prombench v2.4.0` - Compare PR with a specific release version (e.g., from [quay.io/prometheus/prometheus:releaseVersion](https://quay.io/prometheus/prometheus:releaseVersion)).
- `/prombench v2.4.0 --bench.version=@aca1803ccf5d795eee4b0848707eab26d05965cc` - Compare with 2.4.0 release, but use a specific `aca1803ccf5d795eee4b0848707eab26d05965cc` commit on this repository for `./manifests/prombench` resources.
- `/prombench v2.4.0 --bench.version=mybranch` - Compare with 2.4.0 release, but use a specific `mybranch` on this repository for `./manifests/prombench` resources.
- `/prombench v2.4.0 --bench.directory=manifests/prombench-agent-mode` - Compare with 2.4.0 release, but use a specific resource directory on `master` branch for this repository. Currently there is only `./manifests/prombench` available (default), we might add more modes in the future.

**Restarting Tests:**

- `/prombench restart <release_version>`
- `/prombench restart <release_version> --bench.version=... --bench.directory...` 

**Stopping Tests:**

- `/prombench cancel`

**Printing available commands:**

- `/prombench help`

### Building the Docker Image

Build the Docker image with:

```bash
docker build -t prominfra/prombench:master .
```




