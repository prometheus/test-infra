package main

import (
	"context"
	"io/ioutil"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

type ghClient github.Client

const (
	owner = "prometheus"
	repo  = "prometheus"
)

func NewGHClient(oauthFile string) (*ghClient, error) {
	oauth, err := ioutil.ReadFile(oauthFile)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(oauth)},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := ghClient(*github.NewClient(tc))
	return &client, nil
}

func (c *ghClient) CreateComment(prNumber int, commentBody string) error {

	comment := github.PullRequestComment{
		Body: &commentBody,
	}

	_, _, err := c.PullRequests.CreateComment(context.Background(), owner, repo, prNumber, &comment)
	if err != nil {
		return err
	}
	return nil
}
