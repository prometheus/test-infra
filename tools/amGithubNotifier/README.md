# amGithubNotifier - Alertmanager GitHub Webhook Receiver

A simple bridge for receiving Alertmanager alerts and posting comments to github.

It listens at the `/hook` endpoint on port `:8080` by default.

## Usage

> Note: All alerts sent to this amGithubNotifier must have the `prNum` label, `owner` and `repo` are optional.

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

### Adding default and custom templates
amGithubNotifier assumes that the alerts sent by the [alertmanager](https://github.com/prometheus/alertmanager) will have the `alertname` label in the `grouplabels`.

The required `--template-dir-path` flag should be used to specify the path to the templates directory.

The default template should be named `default`. If there's no `default` file in the templates directory it will error out.

If there's a requirement of a custom template for a perticular alert, you can put a file with the same name as of the `alertname` label and then amGithubNotifier will use that template instead of the `default` one.

Usage:
```
./amGithubNotifier --help
```