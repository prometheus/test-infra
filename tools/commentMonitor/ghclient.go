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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-github/v26/github"
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

// TODO: Once go-github starts supporting repository_dispatch event, use it.
// https://github.com/google/go-github/issues/1316
type repositoryDispatchEvent struct {
	EventType     string            `json:"event_type"`
	ClientPayload map[string]string `json:"client_payload"`
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

func (c githubClient) createRepositoryDispatch(ctx context.Context, eventType string, clientPayload map[string]string) error {
	rd := repositoryDispatchEvent{
		EventType:     eventType,
		ClientPayload: clientPayload,
	}
	body, err := json.Marshal(rd)
	if err != nil {
		return err
	}
	httpClt := &http.Client{}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%v/%v/dispatches", c.owner, c.repo)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("token %v", os.Getenv("GITHUB_TOKEN")))
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/vnd.github.everest-preview+json")
	resp, err := httpClt.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("repository_dispatch event could not be triggered")
	}
	return nil
}
