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
	"errors"
	"fmt"
	"strconv"

	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
)

// alertID returns the alert id.
func alertID(msg *webhook.Message) string {
	return fmt.Sprintf("0x%x", msg.GroupKey)
}

// formatIssueCommentBody constructs an issue comment body from alert annotation.
func formatIssueCommentBody(alert template.Alert) (string, error) {
	if description, ok := alert.Annotations["description"]; ok {
		return description, nil
	}
	return "", errors.New("description annotation not found")
}

// getTargetPR returns the "prNum" label.
func getTargetPR(alert template.Alert) (int, error) {
	if prNum, ok := alert.Labels["prNum"]; ok {
		i, err := strconv.Atoi(prNum)
		if err != nil {
			return 0, err
		}
		return i, nil
	}
	return 0, errors.New("prNum label not found")
}

// getTargetRepo returns the "repo" label if exists else returns ghWebhookReceiverConfig.repo
func (g ghWebhookReceiver) getTargetRepo(alert template.Alert) string {
	if repo, ok := alert.Labels["repo"]; ok {
		return repo
	}
	return g.cfg.repo
}

// getTargetOrg returns the "org" label if exists else returns ghWebhookReceiverConfig.org
func (g ghWebhookReceiver) getTargetOrg(alert template.Alert) string {
	if org, ok := alert.Labels["org"]; ok {
		return org
	}
	return g.cfg.org
}
