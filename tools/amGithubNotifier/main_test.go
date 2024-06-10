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
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/common/model"
)

func TestFormatIssueCommentBody(t *testing.T) {
	var alerts template.Alerts

	for i := 1; i <= 3; i++ {
		l := template.KV{
			"alertname":  "brokesomething",
			"prNum":      "1",
			"otherLabel": "foo",
		}
		a := template.KV{
			"description": fmt.Sprintf("This is some alert for pr %v", i),
			"otherAnn":    "foo",
		}
		alert := template.Alert{
			Status:       string(model.AlertFiring),
			Labels:       l,
			Annotations:  a,
			StartsAt:     time.Time{},
			EndsAt:       time.Time{},
			GeneratorURL: "http://prometheus.io?foo=bar&baz=qux",
		}
		alerts = append(alerts, alert)
	}

	cl := template.KV{
		"alertname":  "brokesomething",
		"otherLabel": "foo",
	}
	ca := template.KV{
		"otherAnn": "foo",
	}
	gl := template.KV{"alertname": "fixAlert", "namespace": "default"}
	data := &template.Data{
		Receiver:          "testReceiver",
		Status:            string(model.AlertFiring),
		Alerts:            alerts,
		GroupLabels:       gl,
		CommonLabels:      cl,
		CommonAnnotations: ca,
		ExternalURL:       "http://alertmanager.com",
	}

	msg := &webhook.Message{
		Version:  "4",
		Data:     data,
		GroupKey: "group_key",
	}

	ctx := context.Background()
	cfg := ghWebhookReceiverConfig{dryRun: true}
	client, err := newGhWebhookReceiver(cfg)
	if err != nil {
		t.Errorf("could not create github client")
	}

	alertcomments, err := client.processAlerts(ctx, msg)
	if err != nil {
		log.Printf("failed to handle alert: %v: %v", alertID(msg), err)
		return
	}
	output := []string{
		"This is some alert for pr 1",
		"This is some alert for pr 2",
		"This is some alert for pr 3",
	}
	for i, c := range alertcomments {
		if output[i] != c {
			t.Errorf("Output did not match.\ngot:\n%#v\nwant:\n%#v", c, output[i])
		}
	}
}
