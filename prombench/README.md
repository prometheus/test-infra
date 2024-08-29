# Automated Prometheus E2E Testing and Benchmarking

![Prombench Design](design.svg)

This setup leverages **GitHub Actions** and **Google Kubernetes Engine (GKE)**, but is designed to be extendable to other Kubernetes providers.

## Overview of Manifest Files

The `/manifest` directory contains Kubernetes manifest files:

- **`cluster_gke.yaml`**: Creates the Main Node in GKE.
- **`cluster_eks.yaml`**: Creates the Main Node in EKS.
- **`cluster-infra/`**: Contains persistent components of the Main Node.
- **`prombench/`**: Resources created and destroyed for each Prombench test.

## Setup and Running Prombench

Prombench can be run on different providers. Follow these instructions based on your provider:

- [Google Kubernetes Engine (GKE)](docs/gke.md)
- [Kubernetes In Docker (KIND)](docs/kind.md)
- [Elastic Kubernetes Service (EKS)](docs/eks.md)

## Setting Up GitHub Actions

1. Place a workflow file in the `.github` directory of your repository. Refer to the [Prometheus GitHub repository](https://github.com/prometheus/prometheus) for an example.

2. Create a GitHub Action secret `TEST_INFRA_PROVIDER_AUTH` with the base64 encoded content of the `AUTH_FILE`:

   ```bash
   cat $AUTH_FILE | base64 -w 0
   ```

### Triggering Tests via GitHub Comment

**Starting Tests:**

- `/prombench main` or `/prombench master` - Compare PR with the main/master branch.
- `/prombench v2.4.0` - Compare PR with a specific release version (e.g., from [quay.io/prometheus/prometheus:releaseVersion](https://quay.io/prometheus/prometheus:releaseVersion)).

**Restarting Tests:**

- `/prombench restart <release_version>`

**Stopping Tests:**

- `/prombench cancel`

### Building the Docker Image

Build the Docker image with:

```bash
docker build -t prominfra/prombench:master .
```

