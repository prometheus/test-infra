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
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/go-github/v29/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

type Environment interface {
	BenchFunc() string
	CompareTarget() string

	PostErr(err string) error
	PostResults(cmps []BenchCmp) error

	Repo() *git.Repository
}

type environment struct {
	logger Logger

	benchFunc     string
	compareTarget string
}

func (e environment) BenchFunc() string     { return e.benchFunc }
func (e environment) CompareTarget() string { return e.compareTarget }

type Local struct {
	environment

	repo *git.Repository
}

func newLocalEnv(e environment) (Environment, error) {
	r, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, err
	}
	e.logger.Println("[Local Mode]", "\nBenchmarking current version versus:", e.compareTarget, "\nBenchmark func regex:", e.benchFunc)
	return &Local{environment: e, repo: r}, nil
}

func (l *Local) PostErr(string) error { return nil } // Noop. We will see error anyway.

func (l *Local) PostResults(cmps []BenchCmp) error {
	fmt.Println("Results:")
	Render(os.Stdout, cmps, false, false, l.compareTarget)
	return nil
}

func (l *Local) Repo() *git.Repository { return l.repo }

// TODO: Add unit test(!).
type GitHub struct {
	environment

	repo   *git.Repository
	client *gitHubClient
}

func newGitHubEnv(ctx context.Context, e environment, gc *gitHubClient, workspace string) (Environment, error) {
	r, err := git.PlainCloneContext(ctx, fmt.Sprintf("%s/%s", workspace, gc.repo), false, &git.CloneOptions{
		URL:      fmt.Sprintf("https://github.com/%s/%s.git", gc.owner, gc.repo),
		Progress: os.Stdout,
		Depth:    1,
	})
	if err != nil {
		return nil, errors.Wrap(err, "clone git repository")
	}

	if err := os.Chdir(filepath.Join(workspace, gc.repo)); err != nil {
		return nil, errors.Wrapf(err, "changing to %s/%s dir", workspace, gc.repo)
	}

	g := &GitHub{
		environment: e,
		repo:        r,
		client:      gc,
	}

	wt, err := g.repo.Worktree()
	if err != nil {
		return nil, err
	}

	if err := r.FetchContext(ctx, &git.FetchOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec(fmt.Sprintf("+refs/pull/%d/head:refs/heads/pullrequest", gc.prNumber)),
		},
		Progress: os.Stdout,
	}); err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, errors.Wrap(err, "fetch to pull request branch")
	}

	if err = wt.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName("pullrequest"),
	}); err != nil {
		return nil, errors.Wrap(err, "switch to pull request branch")
	}

	e.logger.Println("[GitHub Mode]", gc.owner, ":", gc.repo, "\nBenchmarking PR -", gc.prNumber, "versus:", e.compareTarget, "\nBenchmark func regex:", e.benchFunc)
	return g, nil
}

func (g *GitHub) PostErr(err string) error {
	if err := g.client.postComment(fmt.Sprintf("%v. Benchmark did not complete, please check action logs.", err)); err != nil {
		return errors.Wrap(err, "posting err")
	}
	return nil
}

func (g *GitHub) PostResults(cmps []BenchCmp) error {
	b := bytes.Buffer{}
	Render(&b, cmps, false, false, g.compareTarget)
	return g.client.postComment(formatCommentToMD(b.String()))
}

func (g *GitHub) Repo() *git.Repository { return g.repo }

type gitHubClient struct {
	owner    string
	repo     string
	prNumber int
	client   *github.Client
	dryrun   bool
}

func newGitHubClient(ctx context.Context, owner, repo string, prNumber int, dryrun bool) (*gitHubClient, error) {
	ghToken, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok && !dryrun {
		return nil, fmt.Errorf("GITHUB_TOKEN missing")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	c := gitHubClient{
		client:   github.NewClient(tc),
		owner:    owner,
		repo:     repo,
		prNumber: prNumber,
		dryrun:   dryrun,
	}
	return &c, nil
}

func (c *gitHubClient) postComment(comment string) error {
	if c.dryrun {
		return nil
	}

	issueComment := &github.IssueComment{Body: github.String(comment)}
	_, _, err := c.client.Issues.CreateComment(context.Background(), c.owner, c.repo, c.prNumber, issueComment)
	return err
}
