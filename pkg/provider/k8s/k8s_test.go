// Copyright The Prometheus Authors
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

package k8s

import "testing"

func TestExtractWave(t *testing.T) {
	tests := []struct {
		fileName  string
		want      int
		wantError bool
	}{
		{"4_fake-webserver.yaml", 4, false},
		{"path/to/5_prometheus-test-pr_deployment.yaml", 5, false},
		{"10_loadgen.yaml", 10, false},
		{"1_namespace.yaml", 1, false},
		{"no_number.yaml", 0, true},
		{"", 0, true},
		{"nodash", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			got, err := extractWave(tt.fileName)
			if tt.wantError {
				if err == nil {
					t.Errorf("extractWave(%q) expected error, got nil", tt.fileName)
				}
				return
			}
			if err != nil {
				t.Errorf("extractWave(%q) unexpected error: %v", tt.fileName, err)
				return
			}
			if got != tt.want {
				t.Errorf("extractWave(%q) = %d, want %d", tt.fileName, got, tt.want)
			}
		})
	}
}
