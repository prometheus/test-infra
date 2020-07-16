# infra: A cli tool to create/scale/delete k8s clusters and deploy manifest files.

Currently it supports GKE, but it is designed in a way that adding more providers should be easy.

### Parsing of files

Files passed to `infra` will be parsed using golang templates, to skip parsing and load the file as is, use `noparse` suffix.

Eg. `somefile.yaml` will be parsed, whereas `somefile_noparse.yaml` will not be parsed.

## Usage and examples:

[embedmd]:# (infra-flags.txt)
```txt
usage: infra [<flags>] <command> [<args> ...]

The prometheus/test-infra deployment tool

Flags:
  -h, --help  Show context-sensitive help (also try --help-long and --help-man).

Commands:
  help [<command>...]
    Show help.

  gke cluster create
    gke cluster create -a service-account.json -f FileOrFolder

  gke cluster delete
    gke cluster delete -a service-account.json -f FileOrFolder

  gke nodepool create
    gke nodepool create -a service-account.json -f FileOrFolder

  gke nodepool delete
    gke nodepool delete -a service-account.json -f FileOrFolder

  gke nodepool check-running
    gke nodepool check-running -a service-account.json -f FileOrFolder

  gke nodepool check-deleted
    gke nodepool check-deleted -a service-account.json -f FileOrFolder

  gke resource apply
    gke resource apply -a service-account.json -f manifestsFileOrFolder -v
    PROJECT_ID:test -v ZONE:europe-west1-b -v CLUSTER_NAME:test -v
    hashStable:COMMIT1 -v hashTesting:COMMIT2

  gke resource delete
    gke resource delete -a service-account.json -f manifestsFileOrFolder -v
    PROJECT_ID:test -v ZONE:europe-west1-b -v CLUSTER_NAME:test -v
    hashStable:COMMIT1 -v hashTesting:COMMIT2

  kind cluster create
    kind cluster create -f File -v PR_NUMBER:$PR_NUMBER -v
    CLUSTER_NAME:$CLUSTER_NAME

  kind cluster delete
    kind cluster delete -f File -v PR_NUMBER:$PR_NUMBER -v
    CLUSTER_NAME:$CLUSTER_NAME

  kind resource apply
    kind resource apply -f manifestsFileOrFolder -v hashStable:COMMIT1 -v
    hashTesting:COMMIT2

  kind resource delete
    kind resource delete -f manifestsFileOrFolder -v hashStable:COMMIT1 -v
    hashTesting:COMMIT2


```

### Building Docker Image

```
docker build -t prominfra/infra:master .
```
