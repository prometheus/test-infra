# amGithubNotifier - Alertmanager GitHub Webhook Receiver

A simple bridge for receiving Alertmanager alerts and posting comments to github.

By default it listens at `/hook` on port `:8080`.

## Usage

> Note: All alerts sent to amGithubNotifier must have the `prNum` label and `description` annotation, `org` and `repo` labels are optional but will take precedence over cli args if provided.

Example `alerts.rules.yml`:
```yaml
groups:
- name: groupname
  rules:
  - alert: alertname
    expr: up == 0
    labels:
      severity: info
      prNum: '{{ $labels.prNum }}'
      org: prometheus
      repo: prombench
    annotations:
      description: 'description of the alert'
```


### Example for building the docker image
From the repository root:
```
$ make docker DOCKERFILE_PATH=prombench/tools/amGithubNotifier/Dockerfile DOCKER_IMAGE_NAME=amgithubnotifier DOCKERBUILD_CONTEXT=prombench/ DOCKER_IMAGE_TAG=0.0.1
```

#### Usage and examples:
```
./amGithubNotifier --help
```