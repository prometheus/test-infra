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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/tools/benchmark/parse"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

// TODO: Add unit test.
type Benchmarker struct {
	logger Logger

	benchFunc      string
	benchTime      time.Duration
	benchTimeout   time.Duration
	resultCacheDir string

	c    *commander
	repo *git.Repository
}

func newBenchmarker(logger Logger, env Environment, c *commander, benchTime time.Duration, benchTimeout time.Duration, resultCacheDir string) *Benchmarker {
	return &Benchmarker{
		logger:         logger,
		benchFunc:      env.BenchFunc(),
		benchTime:      benchTime,
		benchTimeout:   benchTimeout,
		c:              c,
		repo:           env.Repo(),
		resultCacheDir: resultCacheDir,
	}
}

func (b *Benchmarker) benchOutFileName(commit plumbing.Hash) (string, error) {
	// Sanitize bench func.
	bb := bytes.Buffer{}
	e := base64.NewEncoder(base64.StdEncoding, &bb)
	if _, err := e.Write([]byte(b.benchFunc)); err != nil {
		return "", err
	}
	if err := e.Close(); err != nil {
		return "", err
	}

	return fmt.Sprintf("%s-%s.out", bb.String(), commit.String()), nil
}

func (b *Benchmarker) execBenchmark(pkgRoot string, commit plumbing.Hash) (out string, err error) {
	fileName, err := b.benchOutFileName(commit)
	if err != nil {
		return "", err
	}

	if _, err := ioutil.ReadFile(filepath.Join(b.resultCacheDir, fileName)); err == nil {
		fmt.Println("Found previous results for ", fileName, b.benchFunc, "Reusing.")
		return filepath.Join(b.resultCacheDir, fileName), nil
	}

	// TODO(bwplotka): Allow memprofiles.
	// 'go test' flags: https://golang.org/cmd/go/#hdr-Testing_flags
	extraArgs := []string{"-benchtime", b.benchTime.String()}
	extraArgs = append(extraArgs, "-timeout", b.benchTimeout.String())
	benchCmd := []string{"bash", "-c", strings.Join(append(append([]string{"cd", pkgRoot, "&&", "go", "test", "-run", "^$", "-bench", fmt.Sprintf("^%s$", b.benchFunc), "-benchmem"}, extraArgs...), "./..."), " ")}

	b.logger.Println("Executing benchmark command for", commit.String())
	b.logger.Println(benchCmd)
	out, err = b.c.exec(benchCmd...)
	if err != nil {
		return "", errors.Wrap(err, "benchmark ended with an error.")
	}

	fn := filepath.Join(b.resultCacheDir, fileName)
	if b.resultCacheDir != "" {
		if err := os.MkdirAll(b.resultCacheDir, os.ModePerm); err != nil {
			return "", err
		}
	}
	if err := ioutil.WriteFile(fn, []byte(out), os.ModePerm); err != nil {
		return "", err
	}
	return fn, nil
}

func parseFile(path string) (parse.Set, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	bb, err := parse.ParseSet(f)
	if err != nil {
		return nil, err
	}
	return bb, nil
}

func (b *Benchmarker) compareBenchmarks(beforeFile, afterFile string) ([]BenchCmp, error) {
	before, err := parseFile(beforeFile)
	if err != nil {
		return nil, errors.Wrapf(err, "open %s", beforeFile)
	}

	after, err := parseFile(afterFile)
	if err != nil {
		return nil, errors.Wrapf(err, "open %s", afterFile)
	}

	// Use benchcmp library directly.
	// TODO(bwplotka): benchstat is new thing - we might add choice for funcbench to choose from? (:
	cmps, warnings := Correlate(before, after)
	for _, warn := range warnings {
		b.logger.Println(warn)
	}
	if len(cmps) == 0 {
		return nil, errors.New("no repeated benchmarks")
	}

	return cmps, nil
}

func (b *Benchmarker) compareSubBenchmarks(string) ([]BenchCmp, error) {
	// TODO(bwplotka): Implement.
	return nil, errors.New("not implemented")
}

func formatCommentToMD(rawTable string) string {
	tableContent := strings.Split(rawTable, "\n")
	for i := 0; i <= len(tableContent)-1; i++ {
		e := tableContent[i]
		switch {
		case e == "":

		case strings.Contains(e, "old ns/op"):
			e = "| Benchmark | Old ns/op | New ns/op | Delta |"
			tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

		case strings.Contains(e, "old MB/s"):
			e = "| Benchmark | Old MB/s | New MB/s | Speedup |"
			tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

		case strings.Contains(e, "old allocs"):
			e = "| Benchmark | Old allocs | New allocs | Delta |"
			tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

		case strings.Contains(e, "old bytes"):
			e = "| Benchmark | Old bytes | New bytes | Delta |"
			tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

		default:
			// Replace spaces with "|".
			e = strings.Join(strings.Fields(e), "|")
		}
		tableContent[i] = e
	}
	return strings.Join(tableContent, "\n")

}
