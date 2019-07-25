// Copyright 2019 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"strconv"
	"text/template"

	"github.com/prometheus/alertmanager/notify/webhook"
)

// alertID returns the alert id.
func alertID(msg *webhook.Message) string {
	return fmt.Sprintf("0x%x", msg.GroupKey)
}

// formatIssueCommentBody constructs an issue body from a webhook message.
func formatIssueCommentBody(msg *webhook.Message, tmpl alertTemplate) (string, error) {
	var buf bytes.Buffer
	parsedTemplate := template.Must(template.New(tmpl.alertName).Parse(tmpl.templateString))
	err := parsedTemplate.Execute(&buf, msg)
	if err != nil {
		log.Printf("error executing template: %s", err)
		return "", err
	}
	return buf.String(), nil
}

// getTargetPR returns the "prNum" label.
func getTargetPR(msg *webhook.Message) (int, error) {
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
func (g ghWebhookReceiver) getTargetRepo(msg *webhook.Message) string {
	if repo, ok := msg.CommonLabels["repo"]; ok {
		return repo
	}
	return g.cfg.defaultRepo
}

// getTargetOwner returns the "owner" label if exists else returns defaultOwner.
func (g ghWebhookReceiver) getTargetOwner(msg *webhook.Message) string {
	if owner, ok := msg.CommonLabels["owner"]; ok {
		return owner
	}
	return g.cfg.defaultOwner
}

// selectTemplate returns the alertTemplate to use
func (g ghWebhookReceiver) selectTemplate(msg *webhook.Message) alertTemplate {
	alertname := msg.Data.GroupLabels["alertname"]
	for _, alertTmpl := range g.alertTemplates {
		if alertTmpl.alertName == alertname {
			return alertTmpl
		}
	}
	return g.defaultTemplate
}
