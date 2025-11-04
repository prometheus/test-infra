// Copyright 2024 The Prometheus Authors
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
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v2"
)

// Global variables and Prometheus metrics

const max404Errors = 30

var (
	domainName = os.Getenv("DOMAIN_NAME")

	queryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "loadgen",
			Name:      "query_duration_seconds",
			Help:      "Query duration",
			Buckets:   prometheus.LinearBuckets(0.05, 0.1, 20),
		},
		[]string{"prometheus", "group", "expr", "type"},
	)
	queryCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "loadgen",
			Name:      "queries_total",
			Help:      "Total amount of queries",
		},
		[]string{"prometheus", "group", "expr", "type"},
	)
	queryFailCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "loadgen",
			Name:      "failed_queries_total",
			Help:      "Amount of failed queries",
		},
		[]string{"prometheus", "group", "expr", "type"},
	)
)

// Querier struct and methods
type Querier struct {
	target         string
	name           string
	groupID        int
	numberOfErrors int

	interval time.Duration
	queries  []Query
	qtype    string
	start    time.Duration
	end      time.Duration
	step     string
	url      string
}

type Query struct {
	Expr string `yaml:"expr"`
}

type QueryGroup struct {
	Name     string  `yaml:"name"`
	Interval string  `yaml:"interval"`
	Queries  []Query `yaml:"queries"`
	Type     string  `yaml:"type,omitempty"`
	Start    string  `yaml:"start,omitempty"`
	End      string  `yaml:"end,omitempty"`
	Step     string  `yaml:"step,omitempty"`
}

type BucketConfig struct {
	Path    string `yaml:"path"`
	MinTime int64  `yaml:"minTime"`
	MaxTime int64  `yaml:"maxTime"`
}

type bucketState struct {
	bucketConfig *BucketConfig
}

func NewQuerier(groupID int, target, prNumber string, qg QueryGroup) *Querier {
	qtype := qg.Type
	if qtype == "" {
		qtype = "instant"
	}

	start := durationSeconds(qg.Start)
	end := durationSeconds(qg.End)

	url := fmt.Sprintf("http://%s/%s/prometheus-%s/api/v1/query", domainName, prNumber, target)
	if qtype == "range" {
		url = fmt.Sprintf("http://%s/%s/prometheus-%s/api/v1/query_range", domainName, prNumber, target)
	}

	return &Querier{
		target:   target,
		name:     qg.Name,
		groupID:  groupID,
		interval: durationSeconds(qg.Interval),
		queries:  qg.Queries,
		qtype:    qtype,
		start:    start,
		end:      end,
		step:     qg.Step,
		url:      url,
	}
}

// Function to load `minTime` and `maxTime` from bucket-config.yml
func loadBucketConfig() (*BucketConfig, error) {
	filePath := flag.String("bucketconfig-file", "/config/bucket-config.yml", "Path to the bucket configuration file")
	flag.Parse()

	_, err := os.Stat(*filePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", *filePath)
	}

	data, err := os.ReadFile(*filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	var bucketConfig BucketConfig
	err = yaml.Unmarshal(data, &bucketConfig)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML: %w", err)
	}

	return &bucketConfig, nil
}

func setconfig(v *BucketConfig, err error) *bucketState {
	// If there is an error in reading bucket-config.yml file then just return nil.
	if err != nil {
		return nil
	}
	return &bucketState{
		bucketConfig: v,
	}
}

func (q *Querier) run(wg *sync.WaitGroup, timeBound *bucketState) {
	defer wg.Done()
	fmt.Printf("Running querier %s %s for %s\n", q.target, q.name, q.url)
	time.Sleep(20 * time.Second)

	for {
		start := time.Now()
		// If timeBound is not nil, both the "absolute" and "current" blocks will run;
		// otherwise, only the "current" block will execute. This execution pattern is used
		// because if Downloaded block data is present, both the head block and downloaded block
		// need to be processed consecutively.
		runBlockMode := "current"
		for _, query := range q.queries {
			switch runBlockMode {
			case "current":
				q.query(query.Expr, "current", nil)
			case "absolute":
				q.query(query.Expr, "absolute", timeBound)
			}
			if runBlockMode == "current" && timeBound != nil {
				runBlockMode = "absolute"
			} else if timeBound != nil {
				runBlockMode = "current"
			}
		}

		wait := q.interval - time.Since(start)
		if wait > 0 {
			time.Sleep(wait)
		}
	}
}

func (q *Querier) query(expr string, timeMode string, timeBound *bucketState) {
	queryCount.WithLabelValues(q.target, q.name, expr, q.qtype).Inc()
	start := time.Now()

	req, err := http.NewRequest("GET", q.url, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		queryFailCount.WithLabelValues(q.target, q.name, expr, q.qtype).Inc()
		return
	}

	qParams := req.URL.Query()
	qParams.Set("query", expr)
	if q.qtype == "range" {
		//  here query is for current block i.e headblock and its range query.
		if timeMode == "current" {
			qParams.Set("start", fmt.Sprintf("%d", int64(time.Now().Add(-q.start).Unix())))
			qParams.Set("end", fmt.Sprintf("%d", int64(time.Now().Add(-q.end).Unix())))
			qParams.Set("step", q.step)
		} else {
			// here query is for downloaded block and its range query.
			endTime := time.Unix(0, timeBound.bucketConfig.MaxTime*int64(time.Millisecond))
			qParams.Set("start", fmt.Sprintf("%d", int64(endTime.Add(-q.start).Unix())))
			qParams.Set("end", fmt.Sprintf("%d", int64(endTime.Add(-q.end).Unix())))
			qParams.Set("step", q.step)
		}
	} else if timeMode == "absolute" {
		// here query is for downloaded block and its instant query.
		blockinstime := time.Unix(0, timeBound.bucketConfig.MaxTime*int64(time.Millisecond))
		qParams.Set("time", fmt.Sprintf("%d", int64(blockinstime.Unix())))
	}
	// here query is for current block and its instant query i.e no need to specify the instant time.
	req.URL.RawQuery = qParams.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error querying Prometheus: %v", err)
		queryFailCount.WithLabelValues(q.target, q.name, expr, q.qtype).Inc()
		return
	}
	defer resp.Body.Close()

	duration := time.Since(start)
	queryDuration.WithLabelValues(q.target, q.name, expr, q.qtype).Observe(duration.Seconds())

	if resp.StatusCode == http.StatusNotFound {
		log.Printf("WARNING: GroupID#%d: Querier returned 404 for Prometheus instance %s.", q.groupID, q.url)
		q.numberOfErrors++
		if q.numberOfErrors >= max404Errors {
			log.Fatalf("ERROR: GroupID#%d: Querier returned 404 for Prometheus instance %s %d times.", q.groupID, q.url, max404Errors)
		}
	} else if resp.StatusCode != http.StatusOK {
		log.Printf("WARNING: GroupID#%d: Querier returned %d for Prometheus instance %s.", q.groupID, resp.StatusCode, q.url)
	} else {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("GroupID#%d: query %s %s, status=%d, size=%d, duration=%.3f", q.groupID, q.target, expr, resp.StatusCode, len(body), duration.Seconds())
	}
}

func durationSeconds(s string) time.Duration {
	if s == "" {
		return 0
	}
	value, err := model.ParseDuration(s)
	if err != nil {
		log.Fatalf("%s", err.Error())
	}
	return time.Duration(value)
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("unexpected arguments")
		fmt.Println("usage: <load_generator> <namespace> <pr_number>")
		os.Exit(2)
	}
	prNumber := os.Args[2]

	configPath := flag.String("config-file", "/etc/loadgen/config.yaml", "Path to the configuration file")
	flag.Parse()

	configFile, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		return
	}

	var config struct {
		Querier struct {
			Groups []QueryGroup `yaml:"groups"`
		} `yaml:"querier"`
	}
	if err := yaml.Unmarshal(configFile, &config); err != nil {
		log.Fatalf("Failed to parse config: %v", err)
	}

	fmt.Println("Loaded configuration")

	var wg sync.WaitGroup

	bucketConfig, err := loadBucketConfig()
	timeBound := setconfig(bucketConfig, err)

	for i, group := range config.Querier.Groups {
		wg.Add(1)
		go NewQuerier(i, "pr", prNumber, group).run(&wg, timeBound)
		wg.Add(1)
		go NewQuerier(i, "release", prNumber, group).run(&wg, timeBound)
	}

	prometheus.MustRegister(queryDuration, queryCount, queryFailCount)
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		log.Println("Starting HTTP server on :8080")
		log.Fatal(http.ListenAndServe(":8080", nil))
	}()

	wg.Wait()
}
