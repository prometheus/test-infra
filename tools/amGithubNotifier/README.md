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

#### Usage and examples:
[embedmd]:# (amGithubNotifier-flags.txt)
```txt
usage: amGithubNotifier --org=ORG --repo=REPO [<flags>]

alertmanager github webhook receiver

  Example: ./amGithubNotifier --org=prometheus --repo=prometheus --port=8080

  Note: All alerts sent to amGithubNotifier must have the prNum label and description
  annotation, org and repo labels are optional but will take precedence over cli args
  if provided.


Flags:
  --help         Show context-sensitive help (also try --help-long and
                 --help-man).
  --authfile="/etc/github/oauth"
                 path to github oauth token file
  --org=ORG      name of the org
  --repo=REPO    name of the repo
  --port="8080"  port number to run the server in
  --dryrun       dry run for github api

```
### Building Docker Image

From the repository root:

```
$ make docker DOCKERFILE_PATH=tools/amGithubNotifier/Dockerfile DOCKER_IMAGE_NAME=amgithubnotifier DOCKER_IMAGE_TAG=master
```
