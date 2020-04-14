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
	"testing"
)

func TestMarkdownFormatting(t *testing.T) {
	expectedTable := `| Benchmark | Old ns/op | New ns/op | Delta |
|-|-|-|-|
BenchmarkBufferedSeriesIterator-8|15.7|15.6|-0.64%

| Benchmark | Old MB/s | New MB/s | Speedup |
|-|-|-|-|
BenchmarkBufferedSeriesIterator-8|78201671850.73|79045143860.06|1.01x

| Benchmark | Old allocs | New allocs | Delta |
|-|-|-|-|
BenchmarkBufferedSeriesIterator-8|0|0|+0.00%

| Benchmark | Old bytes | New bytes | Delta |
|-|-|-|-|
BenchmarkBufferedSeriesIterator-8|0|0|+0.00%`
	rawTable := `benchmark master ns/op new ns/op delta
BenchmarkBufferedSeriesIterator-8 15.7 15.6 -0.64%

benchmark master MB/s new MB/s speedup
BenchmarkBufferedSeriesIterator-8 78201671850.73 79045143860.06 1.01x

benchmark master allocs new allocs delta
BenchmarkBufferedSeriesIterator-8 0 0 +0.00%

benchmark master bytes new bytes delta
BenchmarkBufferedSeriesIterator-8 0 0 +0.00%`
	formattedTable := formatCommentToMD(rawTable)
	if formattedTable != expectedTable {
		t.Errorf("Output did not match.\ngot:\n%#v\nwant:\n%#v", formattedTable, expectedTable)
	}
}
