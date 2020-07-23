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

import "testing"

func TestExtractCommand(t *testing.T) {
	testCases := []struct {
		commentBody string
		command     string
	}{
		{"\r\n/funcbench master\t", "/funcbench master"},
		{"\r\n/funcbench master\n", "/funcbench master"},
		{"\r\n/funcbench master\r\n", "/funcbench master"},
		{"/prombench master\r\n", "/prombench master"},
		{"/funcbench master .*\t\r\nSomething", "/funcbench master .*"},
		{"command without forwardslash", "command without forwardslash"},
	}
	for _, tc := range testCases {
		command := extractCommand(tc.commentBody)
		if command != tc.command {
			t.Errorf("want %s, got %s", tc.command, command)
		}
	}
}
func TestCheckCommandPrefix(t *testing.T) {
	cmClient := commentMonitorClient{
		prefixes: []commandPrefix{
			{"/funcbench", "help"},
			{"/prombench", "help"},
			{"/somebench", "help"},
		},
	}
	testCases := []struct {
		command string
		valid   bool
	}{
		{"/funcbench master", true},
		{"/somebench master", true},
		{"/querybench master", false},
		{"prombench master", false},
	}
	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			if cmClient.checkCommandPrefix(tc.command) != tc.valid {
				t.Errorf("want %v, got %v", tc.valid, !tc.valid)
			}
		})
	}
}
