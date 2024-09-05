package main

import (
	"fmt"
	"time"
)

func main() {
	startTime := time.Date(2024, 9, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.Add(24 * time.Hour)

	timeStep := 15 * time.Second

	containerMemoryRSS := map[string]int64{
		"nginx": 524288000,
		"redis": 262144000,
	}

	nodeCPUSecondsTotal := map[string]map[string]float64{
		"node1": {
			"user":   0,
			"system": 0,
			"idle":   0,
		},
	}

	codelabAPIRequestsTotal := map[string]int{
		"GET":  0,
		"POST": 0,
	}

	goRoutines := 100
	kubeletRunningPods := 5
	codelabAPIHTTPRequestsInProgress := 0
	codelabAPIRequestsTotalBar := 0

	currentTime := startTime
	for currentTime.Before(endTime) || currentTime.Equal(endTime) {

		containerMemoryRSS["nginx"] += 1024
		containerMemoryRSS["redis"] += 512

		nodeCPUSecondsTotal["node1"]["user"] += 1
		nodeCPUSecondsTotal["node1"]["system"] += 0.5
		nodeCPUSecondsTotal["node1"]["idle"] += 3

		codelabAPIRequestsTotal["GET"] += 10
		codelabAPIRequestsTotal["POST"] += 5

		goRoutines += 1
		kubeletRunningPods += 1
		codelabAPIHTTPRequestsInProgress += 1
		codelabAPIRequestsTotalBar += 1

		timestamp := currentTime.UnixNano() / 1e6

		fmt.Printf("# TYPE container_memory_rss gauge\n")
		fmt.Printf("container_memory_rss{image=\"nginx\"} %d %d\n", containerMemoryRSS["nginx"], timestamp)
		fmt.Printf("container_memory_rss{image=\"redis\"} %d %d\n", containerMemoryRSS["redis"], timestamp)

		fmt.Printf("# TYPE node_cpu_seconds_total counter\n")
		fmt.Printf("node_cpu_seconds_total{instance=\"node1\",mode=\"user\"} %.1f %d\n", nodeCPUSecondsTotal["node1"]["user"], timestamp)
		fmt.Printf("node_cpu_seconds_total{instance=\"node1\",mode=\"system\"} %.1f %d\n", nodeCPUSecondsTotal["node1"]["system"], timestamp)
		fmt.Printf("node_cpu_seconds_total{instance=\"node1\",mode=\"idle\"} %.1f %d\n", nodeCPUSecondsTotal["node1"]["idle"], timestamp)

		fmt.Printf("# TYPE codelab_api_requests_total counter\n")
		fmt.Printf("codelab_api_requests_total{method=\"GET\"} %d %d\n", codelabAPIRequestsTotal["GET"], timestamp)
		fmt.Printf("codelab_api_requests_total{method=\"POST\"} %d %d\n", codelabAPIRequestsTotal["POST"], timestamp)

		fmt.Printf("# TYPE go_goroutines gauge\n")
		fmt.Printf("go_goroutines %d %d\n", goRoutines, timestamp)

		fmt.Printf("# TYPE kubelet_running_pods gauge\n")
		fmt.Printf("kubelet_running_pods %d %d\n", kubeletRunningPods, timestamp)

		fmt.Printf("# TYPE codelab_api_http_requests_in_progress gauge\n")
		fmt.Printf("codelab_api_http_requests_in_progress %d %d\n", codelabAPIHTTPRequestsInProgress, timestamp)

		fmt.Printf("# TYPE codelab_api_requests_total counter\n")
		fmt.Printf("codelab_api_requests_total{method=\"GET\",path=\"/api/bar\",status=\"200\"} %d %d\n", codelabAPIRequestsTotalBar, timestamp)

		currentTime = currentTime.Add(timeStep)
	}

	fmt.Println("# EOF")
}
