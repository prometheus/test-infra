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
	"os"

	"github.com/google/go-github/v29/github"
	"golang.org/x/oauth2"
)

type githubClient struct {
	clt               *github.Client
	owner             string
	repo              string
	pr                int
	author            string
	commentBody       string
	authorAssociation string
	ctx               context.Context
}

func newGithubClient(ctx context.Context, e *github.IssueCommentEvent) (*githubClient, error) {
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		return nil, fmt.Errorf("env var missing")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	return &githubClient{
		clt:               github.NewClient(tc),
		owner:             *e.GetRepo().Owner.Login,
		repo:              *e.GetRepo().Name,
		pr:                *e.GetIssue().Number,
		author:            *e.Sender.Login,
		authorAssociation: *e.GetComment().AuthorAssociation,
		commentBody:       *e.GetComment().Body,
		ctx:               ctx,
	}, nil
}

func (c githubClient) postComment(commentBody string) error {
	issueComment := &github.IssueComment{Body: github.String(commentBody)}
	_, _, err := c.clt.Issues.CreateComment(c.ctx, c.owner, c.repo, c.pr, issueComment)
	return err
}

func (c githubClient) createLabel(labelName string) error {
	benchmarkLabel := []string{labelName}
	_, _, err := c.clt.Issues.AddLabelsToIssue(c.ctx, c.owner, c.repo, c.pr, benchmarkLabel)
	return err
}

func (c githubClient) getLastCommitSHA() (string, error) {
	// https://developer.github.com/v3/pulls/#list-commits-on-a-pull-request
	listops := &github.ListOptions{Page: 1, PerPage: 250}
	l, _, err := c.clt.PullRequests.ListCommits(c.ctx, c.owner, c.repo, c.pr, listops)
	if len(l) == 0 {
		return "", fmt.Errorf("pr does not have a commit")
	}
	return l[len(l)-1].GetSHA(), err
}

func (c githubClient) createRepositoryDispatch(eventType string, clientPayload map[string]string) error {
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
	_, _, err = c.clt.Repositories.Dispatch(c.ctx, c.owner, c.repo, rd)
	return err
}
