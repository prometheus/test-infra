package main

import (
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	n = flag.Int("port-count", 5, "Number of sequential ports to serve metrics on, starting at 8080.")

	start = time.Now()
)

func main() {
	flag.Parse()

	for i := 0; i < *n; i++ {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{},
		))
		go http.ListenAndServe(fmt.Sprintf(":%d", 8080+i), mux)
	}

	runClient()
}
