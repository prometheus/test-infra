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

package internal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v89/github"
)

type EventDetails struct {
	Owner             string
	Repo              string
	PR                int
	Author            string
	AuthorAssociation string
}

func NewEventDetails(e *github.IssueCommentEvent) EventDetails {
	return EventDetails{
		Owner:             *e.GetRepo().Owner.Login,
		Repo:              *e.GetRepo().Name,
		PR:                *e.GetIssue().Number,
		Author:            *e.Sender.Login,
		AuthorAssociation: e.Comment.GetAuthorAssociation(),
	}
}

type GithubClient struct {
	clt *github.Client
	ctx context.Context

	EventDetails
}

func NewGithubClient(ctx context.Context, token string, issueDetails EventDetails) (*GithubClient, error) {
	ghClient, err := github.NewClient(github.WithAuthToken(token))
	if err != nil {
		return nil, err
	}
	return &GithubClient{
		clt: ghClient,
		ctx: ctx,

		EventDetails: issueDetails,
	}, nil
}

func (c *GithubClient) GetLastCommitSHA() (string, error) {
	// https://developer.github.com/v3/pulls/#list-commits-on-a-pull-request
	listops := &github.ListOptions{Page: 1, PerPage: 250}
	l, _, err := c.clt.PullRequests.ListCommits(c.ctx, c.Owner, c.Repo, c.PR, listops)
	if err != nil {
		return "", fmt.Errorf("ListCommits(%q,%q,%d): %w", c.Owner, c.Repo, c.PR, err)
	}
	if len(l) == 0 {
		return "", fmt.Errorf("pr does not have a commit")
	}
	return l[len(l)-1].GetSHA(), nil
}

func (c *GithubClient) Dispatch(eventType string, args map[string]string) error {
	allArgs, err := json.Marshal(args)
	if err != nil {
		return fmt.Errorf("%w: could not encode client payload", err)
	}
	cp := json.RawMessage(allArgs)

	rd := github.DispatchRequestOptions{
		EventType:     eventType,
		ClientPayload: &cp,
	}

	_, _, err = c.clt.Repositories.Dispatch(c.ctx, c.Owner, c.Repo, rd)
	return err
}

func (c *GithubClient) PostComment(commentBody string) error {
	issueComment := &github.IssueComment{Body: github.Ptr(commentBody)}
	_, _, err := c.clt.Issues.CreateComment(c.ctx, c.Owner, c.Repo, c.PR, issueComment)
	return err
}

func (c *GithubClient) PostLabel(label string) error {
	benchmarkLabel := []string{label}
	if _, _, err := c.clt.Issues.AddLabelsToIssue(c.ctx, c.Owner, c.Repo, c.PR, benchmarkLabel); err != nil {
		return fmt.Errorf("%w : couldn't set label", err)
	}
	return nil
}
