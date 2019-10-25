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
	eventType     string
	clientPayload map[string]string
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
		eventType:     eventType,
		clientPayload: clientPayload,
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
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("token %v", os.Getenv("GITHUB_TOKEN")))
	req.Header.Set("Accept", "application/vnd.github.everest-preview+json")
	_, err = httpClt.Do(req)
	if err != nil {
		return err
	}
	return nil
}
