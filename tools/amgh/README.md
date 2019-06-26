# amgh - Alertmanager GitHub Reciever

A simple webhook server that can be added as a webhook reciver in alertmanager config.

It listens at the `/hook` endpoint and on port `:8080` by default.

### Features
- Posts comments on Github PRs/issues based on the `prNum` label of alerts.
- Optionally specify `owner` and `repo` as alert labels which override the command line flags for individual alerts.

## Usage

> Note: All alerts sent to this amgh must have the `prNum` label, `owner` and `repo` are optional.

Example `alerts.rules.yml`:
```yaml
groups:
- name: groupname
  rules:
  - alert: alertname
    expr: up == 0
    for: 2m
    labels:
      severity: average
      prNum: '{{ $labels.prNum }}'
      owner: prometheus
      repo: prombench
    annotations:
      description: ''
      summary: ''
```
Usage:
```
./amgh --help
```