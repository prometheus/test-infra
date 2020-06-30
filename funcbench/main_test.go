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
	"testing"

	fixtures "github.com/go-git/go-git-fixtures/v4"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

func TestGetTargetInfo(t *testing.T) {
	f := fixtures.Basic().One()
	sto := filesystem.NewStorage(f.DotGit(), cache.NewObjectLRUDefault())
	r, err := git.Open(sto, f.DotGit())
	if err != nil {
		t.Errorf("error when open repository: %s", err)
	}

	testCases := map[string]string{
		"notFound": plumbing.ZeroHash.String(),
		"HEAD":     "6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
		"master":   "6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
		"branch":   "e8d3ffab552895c19b9fcf7aa264d277cde33881",
		"v1.0.0":   "6ecf0ef2c2dffb796033e5a02219af86ec6584e5",
		"918c48b83bd081e863dbe1b80f8998f058cd8294": "918c48b83bd081e863dbe1b80f8998f058cd8294",
	}

	for target, hash := range testCases {
		commit := getTargetInfo(r, target)
		if commit.String() != hash {
			t.Errorf("error when get target %s, expect %s, got %s", target, hash, commit)
		}
	}
}
