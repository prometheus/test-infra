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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/perf/benchstat"
)

func TestFormatMarkdown(t *testing.T) {
	expected := `Benchmark|Old time/op|New time/op|Delta
-|-|-|-
Respond-4|1.69ms ± 0%|1.75ms ± 0%|~ (p=1.000 n=1+1)
RangeQuery/expr=abs(a_one),steps=1000-4|458µs ± 0%|456µs ± 0%|~ (p=1.000 n=1+1)
Parse/expfmt-text/promtestdata.nometa.txt-4|2.39µs ± 0%|2.37µs ± 0%|~ (p=1.000 n=1+1)

Benchmark|Old alloc/op|New alloc/op|Delta
-|-|-|-
Respond-4|241kB ± 0%|233kB ± 0%|~ (p=1.000 n=1+1)
RangeQuery/expr=abs(a_one),steps=1000-4|41.4kB ± 0%|41.4kB ± 0%|~ (p=1.000 n=1+1)
Parse/expfmt-text/promtestdata.nometa.txt-4|921B ± 0%|922B ± 0%|~ (p=1.000 n=1+1)

Benchmark|Old allocs/op|New allocs/op|Delta
-|-|-|-
Respond-4|10.0 ± 0%|9.0 ± 0%|~ (p=1.000 n=1+1)
RangeQuery/expr=abs(a_one),steps=1000-4|1.18k ± 0%|1.19k ± 0%|~ (p=1.000 n=1+1)
Parse/expfmt-text/promtestdata.nometa.txt-4|24.0 ± 0%|24.0 ± 0%|~ (all equal)

Benchmark|Old speed|New speed|Delta
-|-|-|-
Parse/expfmt-text/promtestdata.nometa.txt-4|13.2TB/s ± 0%|11.3TB/s ± 0%|~ (p=1.000 n=1+1)`
	file1 := `BenchmarkRespond-4           710       1691189 ns/op      241368 B/op         10 allocs/op
BenchmarkRangeQuery/expr=abs(a_one),steps=1000-4                                            2310        457700 ns/op       41378 B/op       1182 allocs/op
BenchmarkParse/expfmt-text/promtestdata.nometa.txt-4                                  510378          2388 ns/op    13161439.49 MB/s         921 B/op         24 allocs/op`
	file2 := `BenchmarkRespond-4           688       1751880 ns/op      232637 B/op          9 allocs/op
BenchmarkRangeQuery/expr=abs(a_one),steps=1000-4                                            2553        456152 ns/op       41442 B/op       1186 allocs/op
BenchmarkParse/expfmt-text/promtestdata.nometa.txt-4                                  434798          2374 ns/op    11280285.67 MB/s         922 B/op         24 allocs/op`
	c := &benchstat.Collection{}

	c.AddConfig("file1", []byte(file1))
	c.AddConfig("file2", []byte(file2))

	tables := c.Tables()
	var buf bytes.Buffer
	_ = formatMarkdown(&buf, tables)
	out := buf.String()
	if strings.Compare(expected, strings.TrimSpace(out)) != 0 {
		t.Errorf("Expected:\n%s, but got:\n%s", expected, out)
	}
}

func TestResultIsEmpty(t *testing.T) {
	file1 := `
ok  	github.com/prometheus/prometheus/tsdb/fileutil	0.323s
PASS
BenchmarkIsolation/10-8	445276	2478 ns/op	0 B/op	0 allocs/op
`
	file2 := `
BenchmarkIsolation/100-8	39044	28747 ns/op	6 B/op	0 allocs/op
`
	dir, err := os.MkdirTemp("", "test_empty_result")
	if err != nil {
		t.Error(err)
	}

	files := map[string]string{
		"file1": file1,
		"file2": file2,
	}

	names := make([]string, 0)
	for name, content := range files {
		f := filepath.Join(dir, name)
		if err := os.WriteFile(f, []byte(content), 0644); err != nil {
			t.Error(err)
			return
		}
		names = append(names, f)
	}

	if _, err := compareBenchmarks(names...); err == nil || !strings.Contains(err.Error(), "match any") {
		t.Error("Should return an error indicated that no matching benchmarks found.")
	}
}
