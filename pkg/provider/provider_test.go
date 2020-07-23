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

package provider

import (
	"reflect"
	"testing"
)

func TestMergeDeploymentVars(t *testing.T) {
	dv1 := map[string]string{
		"foo": "apple",
		"bar": "orange",
	}
	dv2 := map[string]string{
		"foo":  "mango",
		"baz":  "banana",
		"buzz": "jackfruit",
	}
	dv3 := map[string]string{
		"foo": "grape",
		"baz": "blueberry",
	}
	testCases := []struct {
		vars   []map[string]string
		merged map[string]string
	}{
		{
			vars:   []map[string]string{dv1, dv2, dv3},
			merged: map[string]string{"bar": "orange", "baz": "blueberry", "buzz": "jackfruit", "foo": "grape"},
		},
		{
			vars:   []map[string]string{dv3, dv2, dv1},
			merged: map[string]string{"bar": "orange", "baz": "banana", "buzz": "jackfruit", "foo": "apple"},
		},
		{
			vars:   []map[string]string{dv3, dv1, dv2},
			merged: map[string]string{"bar": "orange", "baz": "banana", "buzz": "jackfruit", "foo": "mango"},
		},
	}

	for _, tc := range testCases {
		r := MergeDeploymentVars(tc.vars...)
		if eq := reflect.DeepEqual(tc.merged, r); !eq {
			t.Errorf("\nexpect %#v\ngot %#v", tc.merged, r)
		}
	}
}
