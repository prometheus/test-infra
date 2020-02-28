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
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/go-github/v29/github"
)

type githubClient struct {
	clt               *github.Client
	owner             string
	repo              string
	pr                int
	author            string
	commentBody       string
	authorAssociation string
}

func (c githubClient) postComment(ctx context.Context, commentBody string) error {
	issueComment := &github.IssueComment{Body: github.String(commentBody)}
	_, _, err := c.clt.Issues.CreateComment(ctx, c.owner, c.repo, c.pr, issueComment)
	return err
}

func (c githubClient) createLabel(ctx context.Context, labelName string) error {
	benchmarkLabel := []string{labelName}
	_, _, err := c.clt.Issues.AddLabelsToIssue(ctx, c.owner, c.repo, c.pr, benchmarkLabel)
	return err
}

func (c githubClient) getLastCommitSHA(ctx context.Context) (string, error) {
	// https://developer.github.com/v3/pulls/#list-commits-on-a-pull-request
	listops := &github.ListOptions{Page: 1, PerPage: 250}
	l, _, err := c.clt.PullRequests.ListCommits(ctx, c.owner, c.repo, c.pr, listops)
	return l[len(l)-1].GetSHA(), err
}

func (c githubClient) createRepositoryDispatch(ctx context.Context, eventType string, clientPayload map[string]string) error {
	allArgs, err := json.Marshal(clientPayload)
	if err != nil {
		return fmt.Errorf("%v: could not encode client payload", err)
	}
	cp := json.RawMessage(string(allArgs))

	rd := github.DispatchRequestOptions{
		EventType:     eventType,
		ClientPayload: &cp,
	}

	log.Printf("creating repository_dispatch with payload: %v", string(allArgs))
	_, _, err = c.clt.Repositories.Dispatch(ctx, c.owner, c.repo, rd)
	return err
}
