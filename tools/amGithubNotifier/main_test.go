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
	"log"
	"testing"
	"time"

	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/common/model"
)

func TestFormatIssueCommentBody(t *testing.T) {
	const testTemplateString = `
### {{ index .Data.GroupLabels "alertname" }}:{{ index .Data.GroupLabels "namespace" }} [{{len .Data.Alerts}}]

Alertmanager URL: {{.Data.ExternalURL}}

---
{{range .Data.Alerts}}
<details>
<summary> {{if eq .Status "firing"}}🔥 {{ else }} ✅ {{end}} {{.Status}} | {{index .Labels "node"}}</summary>

**Explore Alert:** [prometheus explorer]({{.GeneratorURL}})

{{if .Labels}} **Labels:** {{- end}}

{{range $key, $_ := .Labels}} {{ $key }} | {{- end }}
{{range $_, $_ := .Labels}} --- | {{- end }}
{{range $_, $value := .Labels}} {{ $value }} | {{- end }}

{{if .Annotations}} **Annotations:** {{- end}}
{{range $key, $value := .Annotations}}
- **{{$key}}** : {{$value -}}
</details>{{end}}{{end}}`
	testAlertTemplate := alertTemplate{
		alertName:      "testTemplate",
		templateString: testTemplateString,
	}

	cl := template.KV{
		"testLabel":  "labelData",
		"testLabel2": "labelData",
		"node":       "testNodeName",
	}
	ca := template.KV{"testAnn": "annData"}
	alert1 := template.Alert{
		Status:       string(model.AlertFiring),
		Labels:       cl,
		Annotations:  ca,
		StartsAt:     time.Time{},
		EndsAt:       time.Time{},
		GeneratorURL: "http://www.prom.io?foo=bar&baz=qux",
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
		ExternalURL:       "http://alertmanager.com",
	}

	msg := &webhook.Message{
		Version:  "4",
		Data:     data,
		GroupKey: "group_key",
	}
	body, err := formatIssueCommentBody(msg, testAlertTemplate)
	if err != nil {
		log.Fatalf("%v", err)
	}
	output := `
### fixAlert:default [1]

Alertmanager URL: http://alertmanager.com

---

<details>
<summary> 🔥  firing | testNodeName</summary>

**Explore Alert:** [prometheus explorer](http://www.prom.io?foo=bar&baz=qux)

 **Labels:**

 node | testLabel | testLabel2 |
 --- | --- | --- |
 testNodeName | labelData | labelData |

 **Annotations:**

- **testAnn** : annData</details>`
	if body != output {
		t.Errorf("Output did not match.\ngot:\n%#v\nwant:\n%#v", body, output)
	}

}