package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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
		registry.MustRegister(prometheus.NewProcessCollector(os.Getpid(), ""))
	}
	if *registerGoMetrics {
		registry.MustRegister(prometheus.NewGoCollector())
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
		go http.ListenAndServe(fmt.Sprintf(":%d", 8080+i), mux)
	}

	runClient()
}
