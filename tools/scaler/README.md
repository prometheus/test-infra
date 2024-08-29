# Kubernetes Deployment Scaler

A CLI tool designed to scale Kubernetes deployments up and down periodically from within a Kubernetes cluster. It utilizes the [Kubernetes provider](../../pkg/provider/k8s) to manage scaling operations.

## Table of Contents

1. [RBAC Roles](#rbac-roles)
2. [Usage](#usage)
   - [Sample Command](#sample-command)
   - [Command Flags](#command-flags)
   - [Arguments](#arguments)
3. [Building Docker Image](#building-docker-image)

## RBAC Roles

To use this tool, ensure the container running it has the appropriate RBAC configuration. Below is an example configuration:

```yaml
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: scaler
  namespace: prombench
rules:
- apiGroups: ["apps"]   # apiVersion of the deployment being scaled
  resources:
  - deployments
  verbs: ["get", "list", "update"]
```

## Usage

This CLI tool is meant to be executed inside a Kubernetes container. Below is an example of how to use the tool, along with the available flags and arguments.

### Sample Command

```bash
./scaler scale --file=deployment.yaml <max> <min> <interval>
```

**Example**:
```bash
./scaler scale -v NAMESPACE:scale -f fake-webserver.yaml 20 1 15m
```

### Command Flags

- `-h, --help`: Displays context-sensitive help (also try `--help-long` and `--help-man`).
- `-f, --file=FILE ...`: Specifies the YAML file or folder that describes the parameters for the deployment.
- `-v, --vars=VARS ...`: Substitutes the token holders in the YAML file with provided values. Follows standard Go template formatting (e.g., `{{ .hashStable }}`).

### Arguments

- `<max>`: Number of replicas to scale up to.
- `<min>`: Number of replicas to scale down to.
- `<interval>`: Time to wait before changing the number of replicas (e.g., `15m` for 15 minutes).

## Building Docker Image

To build the Docker image for the scaler tool, execute the following command:

```bash
docker build -t prominfra/scaler:master .
```
