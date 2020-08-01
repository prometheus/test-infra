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
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-github/v29/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"golang.org/x/perf/benchstat"
)

type Environment interface {
	BenchFunc() string
	CompareTarget() string
	SetHashStrings(compareTargetHash, repoHeadHashString string)

	PostErr(err string) error
	PostResults(tables []*benchstat.Table, extraInfo ...string) error

	Repo() *git.Repository
}

type environment struct {
	logger Logger

	benchFunc               string
	compareTarget           string
	compareTargetHashString string
	repoHeadHashString      string
}

func (e environment) BenchFunc() string     { return e.benchFunc }
func (e environment) CompareTarget() string { return e.compareTarget }
func (e *environment) SetHashStrings(compareTargetHash, repoHeadHashString string) {
	e.compareTargetHashString = compareTargetHash
	e.repoHeadHashString = repoHeadHashString
}

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

func (l *Local) PostResults(tables []*benchstat.Table, extraInfo ...string) error {
	legend := fmt.Sprintf("Old: %s\nNew: %s",
		l.compareTargetHashString,
		l.repoHeadHashString,
	)
	fmt.Printf("Results:\n%s\n", legend)

	var buf bytes.Buffer
	benchstat.FormatText(&buf, tables)

	os.Stdout.Write(buf.Bytes())

	return nil
}

func (l *Local) Repo() *git.Repository { return l.repo }

// TODO: Add unit test(!).
type GitHub struct {
	environment

	repo   *git.Repository
	client *gitHubClient

	ctx context.Context
}

func newGitHubEnv(ctx context.Context, e environment, gc *gitHubClient, workspace string) (Environment, error) {

	var r *git.Repository
	var err error
	// Retry 10 times.
	for i := 1; i <= 10; i++ {
		if err := os.RemoveAll(filepath.Join(workspace, gc.repo)); err != nil {
			return nil, err
		}
		e.logger.Println("Cloning ", gc.owner, ":", gc.repo, " is in progress. Checking in ", 10)
		time.Sleep(10 * time.Second)
		r, err = git.PlainCloneContext(ctx, filepath.Join(workspace, gc.repo), false, &git.CloneOptions{
			URL:      fmt.Sprintf("https://github.com/%s/%s.git", gc.owner, gc.repo),
			Progress: os.Stdout,
		})
		if err == nil {
			break
		}
	}
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
		ctx:         ctx,
	}

	if err := os.Setenv("CGO_ENABLED", "0"); err != nil {
		return nil, err
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

func (g *GitHub) PostErr(txt string) error {
	c := fmt.Sprintf(
		"Old: `%s`\nNew: `PR-%d`\n%s",
		g.compareTarget,
		g.client.prNumber,
		txt,
	)

	if err := g.client.postComment(c); err != nil {
		return err
	}

	return nil
}

func (g *GitHub) PostResults(tables []*benchstat.Table, extraInfo ...string) error {
	b := bytes.Buffer{}
	if err := formatMarkdown(&b, tables); err != nil {
		return err
	}

	legend := fmt.Sprintf("Old: `%s`/`%s`\nNew: `PR-%d`/`%s`",
		g.compareTarget,
		g.compareTargetHashString,
		g.client.prNumber,
		g.repoHeadHashString,
	)
	result := fmt.Sprintf(
		"<details><summary>Click to check benchmark result</summary>\n\n%s\n%s\n%s</details>",
		legend,
		strings.Join(extraInfo, "\n"),
		b.String(),
	)
	return g.client.postComment(result)
}

func (g *GitHub) Repo() *git.Repository { return g.repo }

type gitHubClient struct {
	owner     string
	repo      string
	prNumber  int
	client    *github.Client
	nocomment bool
	ctx       context.Context
}

func newGitHubClient(ctx context.Context, owner, repo string, prNumber int, nocomment bool) (*gitHubClient, error) {
	ghToken, ok := os.LookupEnv("GITHUB_TOKEN")
	if !ok && !nocomment {
		return nil, fmt.Errorf("GITHUB_TOKEN missing")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	c := gitHubClient{
		client:    github.NewClient(tc),
		owner:     owner,
		repo:      repo,
		prNumber:  prNumber,
		nocomment: nocomment,
		ctx:       ctx,
	}
	return &c, nil
}

func (c *gitHubClient) postComment(comment string) error {
	if c.nocomment {
		return nil
	}

	issueComment := &github.IssueComment{Body: github.String(comment)}
	_, _, err := c.client.Issues.CreateComment(c.ctx, c.owner, c.repo, c.prNumber, issueComment)
	return err
}
