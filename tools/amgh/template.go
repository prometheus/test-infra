package main

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log"
	"strconv"

	"github.com/prometheus/alertmanager/notify"
)

const alertMD = `
## Alertname: {{ index .Data.GroupLabels "alertname" }}
Alertmanager URL: {{.Data.ExternalURL}}
{{range .Data.Alerts}}
  * {{.Status}} {{.GeneratorURL}}
  {{if .Labels}}
    Labels:
  {{- end}}
  {{range $key, $value := .Labels}}
    - {{$key}} = {{$value -}}
  {{end}}
  {{if .Annotations}}
    Annotations:
  {{- end}}
  {{range $key, $value := .Annotations}}
    - {{$key}} = {{$value -}}
  {{end}}
{{end}}
`

var alertTemplate = template.Must(template.New("alert").Parse(alertMD))

// alertID returns the alert id.
func alertID(msg *notify.WebhookMessage) string {
	return fmt.Sprintf("0x%x", msg.GroupKey)
}

// formatIssueBody constructs an issue body from a webhook message.
func formatIssueBody(msg *notify.WebhookMessage) (string, error) {
	var buf bytes.Buffer
	err := alertTemplate.Execute(&buf, msg)
	if err != nil {
		log.Printf("error executing template: %s", err)
		return "", err
	}
	return buf.String(), nil
}

// getTargetPR returns the "prNum" label.
func getTargetPR(msg *notify.WebhookMessage) (int, error) {
	if prNum, ok := msg.CommonLabels["prNum"]; ok {
		i, err := strconv.Atoi(prNum)
		if err != nil {
			return 0, err
		}
		return i, nil
	}
	return 0, errors.New("prNum label not found")
}

// getTargetRepo returns the "repo" label if exists else returns defaultRepo.
func (g ghWebhookReciever) getTargetRepo(msg *notify.WebhookMessage) string {
	if repo, ok := msg.CommonLabels["repo"]; ok {
		return repo
	}
	return g.cfg.defaultRepo
}

// getTargetOwner returns the "owner" label if exists else returns defaultOwner.
func (g ghWebhookReciever) getTargetOwner(msg *notify.WebhookMessage) string {
	if owner, ok := msg.CommonLabels["owner"]; ok {
		return owner
	}
	return g.cfg.defaultOwner
}
