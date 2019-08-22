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
	"math"
	"time"
)

var oscillationPeriod = flag.Duration("oscillation-period", 5*time.Minute, "The duration of the rate oscillation period.")

func runClient() {
	oscillationFactor := func() float64 {
		return 2 + math.Sin(math.Sin(2*math.Pi*float64(time.Since(start))/float64(*oscillationPeriod)))
	}

	// GET /api/foo.
	go func() {
		for {
			handleAPI("GET", "/api/foo")
			time.Sleep(time.Duration(3*oscillationFactor()) * time.Millisecond)
		}
	}()
	// POST /api/foo.
	go func() {
		for {
			handleAPI("POST", "/api/foo")
			time.Sleep(time.Duration(25*oscillationFactor()) * time.Millisecond)
		}
	}()
	// GET /api/bar.
	go func() {
		for {
			handleAPI("GET", "/api/bar")
			time.Sleep(time.Duration(10*oscillationFactor()) * time.Millisecond)
		}
	}()
	// POST /api/bar.
	go func() {
		for {
			handleAPI("POST", "/api/bar")
			time.Sleep(time.Duration(5*oscillationFactor()) * time.Millisecond)
		}
	}()
	// GET /api/baz.
	go func() {
		for {
			handleAPI("POST", "/api/baz")
			time.Sleep(time.Duration(70*oscillationFactor()) * time.Millisecond)
		}
	}()
	// GET /api/boom.
	go func() {
		for {
			handleAPI("GET", "/api/boom")
			time.Sleep(time.Duration(80*oscillationFactor()) * time.Millisecond)
		}
	}()
	// GET /api/nonexistent.
	go func() {
		for {
			handleAPI("POST", "/api/boom")
			time.Sleep(time.Duration(90*oscillationFactor()) * time.Millisecond)
		}
	}()

	select {}
}
