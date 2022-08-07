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
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	n = flag.Int(
		"port-count", 5,
		"Number of sequential ports to serve metrics on, starting at 8080.",
	)
	registerProcessMetrics = flag.Bool(
		"enable-process-metrics", true,
		"Include (potentially expensive) process_* metrics.",
	)
	registerGoMetrics = flag.Bool(
		"enable-go-metrics", true,
		"Include (potentially expensive) go_* metrics.",
	)
	allowCompression = flag.Bool(
		"allow-metrics-compression", true,
		"Allow gzip compression of metrics.",
	)

	start = time.Now()
)

func main() {
	flag.Parse()

	if *registerProcessMetrics {
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}
	if *registerGoMetrics {
		registry.MustRegister(collectors.NewGoCollector())
	}

	for i := 0; i < *n; i++ {
		mux := http.NewServeMux()
		mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		mux.Handle("/metrics", promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{
				DisableCompression: !*allowCompression,
			},
		))
		go func(i int) {
			log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", 8080+i), mux))
		}(i)
	}

	runClient()
}
