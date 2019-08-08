# amGithubNotifier - Alertmanager GitHub Webhook Receiver

A simple bridge for receiving Alertmanager alerts and posting comments to github.

By default it listens at `/hook` on port `:8080`.

## Usage

> Note: All alerts sent to amGithubNotifier must have the `prNum` label and `description` annotation, `owner` and `repo` labels are optional but will take precedence over cli args if provided.

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
      description: 'description of the alert'
```


#### Usage and examples:
```
./amGithubNotifier --help
```