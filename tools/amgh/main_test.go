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
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/common/model"
)

func TestFormatIssueCommentBody(t *testing.T) {
	cl := template.KV{"testLabel": "labelData"}
	ca := template.KV{"testAnn": "annData"}
	alert1 := template.Alert{
		Status:       string(model.AlertFiring),
		Labels:       cl,
		Annotations:  ca,
		StartsAt:     time.Time{},
		EndsAt:       time.Time{},
		GeneratorURL: "http://prometheus.io",
	}
	alerts := template.Alerts{
		alert1,
	}
	gl := template.KV{"alertname": "fixAlert", "namespace": "default"}
	data := &template.Data{
		Receiver:          "testReceiver",
		Status:            string(model.AlertFiring),
		Alerts:            alerts,
		GroupLabels:       gl,
		CommonLabels:      cl,
		CommonAnnotations: ca,
		ExternalURL:       "http://somepath.com",
	}

	msg := &notify.WebhookMessage{
		Version:  "4",
		Data:     data,
		GroupKey: "group_key",
	}
	body, err := formatIssueBody(msg)
	if err != nil {
		log.Fatalln("s")
	}
	fmt.Printf("%v\n", body)
}
