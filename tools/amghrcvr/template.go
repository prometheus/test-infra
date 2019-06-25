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

TODO: add graph url from annotations.
`

var alertTemplate = template.Must(template.New("alert").Parse(alertMD))

// id returns the alert id
func id(msg *notify.WebhookMessage) string {
	return fmt.Sprintf("0x%x", msg.GroupKey)
}

// formatTitle constructs an issue title from a webhook message.
func formatTitle(msg *notify.WebhookMessage) string {
	return fmt.Sprintf("%s", msg.Data.GroupLabels["alertname"])
}

// formatIssueBody constructs an issue body from a webhook message.
func formatIssueBody(msg *notify.WebhookMessage) (string, error) {
	var buf bytes.Buffer
	err := alertTemplate.Execute(&buf, msg)
	if err != nil {
		log.Printf("Error executing template: %s", err)
		return "", err
	}
	s := buf.String()
	// do we need the alert id in the comment? i dont think so
	return fmt.Sprintf("<!-- ID: %s -->\n%s", id(msg), s), nil
}

// getTargetRepo returns the "repo" label if exists else returns defaultRepo
func getTargetRepo(msg *notify.WebhookMessage) string {
	if repo, ok := msg.CommonLabels["repo"]; ok {
		return repo
	}
	return defaultRepo
}

// getTargetOwner returns the "owner" label if exists else returns defaultOrg
func getTargetOwner(msg *notify.WebhookMessage) string {
	if owner, ok := msg.CommonLabels["owner"]; ok {
		return owner
	}
	return defaultOwner
}

// getTargetPR returns the "prNum" label
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
