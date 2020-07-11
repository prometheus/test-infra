// Copyright 2020 The Prometheus Authors
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
	"strings"
	"text/template"

	"golang.org/x/perf/benchstat"
)

var renderTemplate = template.Must(template.New("").Funcs(renderFuncs).Parse(`
{{- range $i, $table := . }}
Benchmark|Old {{.Metric}}|New {{.Metric}}{{if .OldNewDelta}}|Delta{{end}}
-|-|-{{if .OldNewDelta}}|-{{end}}

	{{- range $group := group $table.Rows }}
		{{- range $row := . }}
{{ .Benchmark }}{{range .Metrics}}|{{.Format $row.Scaler}}{{end}}{{if $table.OldNewDelta}}|{{replace .Delta "-" "−" -1}} {{.Note}}{{ end }}
		{{- end }}
	{{- end }}
{{ end }}`))

var renderFuncs = template.FuncMap{
	"replace": strings.Replace,
	"group":   formGroup,
}

func formGroup(rows []*benchstat.Row) (out [][]*benchstat.Row) {
	var group string
	var cur []*benchstat.Row
	for _, r := range rows {
		if r.Group != group {
			group = r.Group
			if len(cur) > 0 {
				out = append(out, cur)
				cur = nil
			}
		}
		cur = append(cur, r)
	}
	if len(cur) > 0 {
		out = append(out, cur)
	}
	return
}

func formatMarkdown(buf *bytes.Buffer, tables []*benchstat.Table) error {
	return renderTemplate.Execute(buf, tables)
}
