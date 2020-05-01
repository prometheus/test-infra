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
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/oklog/run"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

type Logger interface {
	Println(v ...interface{})
}

type logger struct {
	*log.Logger

	verbose bool
}

func (l *logger) FatalError(err error) {
	if l.verbose {
		l.Fatalf("%+v", err)
	}
	l.Fatalf("%v", err)
}

func main() {
	cfg := struct {
		verbose        bool
		nocomment      bool
		owner          string
		repo           string
		resultsDir     string
		workspaceDir   string
		ghPR           int
		benchTime      time.Duration
		benchTimeout   time.Duration
		compareTarget  string
		benchFuncRegex string
	}{}

	app := kingpin.New(
		filepath.Base(os.Args[0]),
		`Benchmark and compare your Go code between sub benchmarks or commits.
		* For BenchmarkFuncName, compare current with master: ./funcbench -v master BenchmarkFuncName
		* For BenchmarkFunc.*, compare current with master: ./funcbench -v master BenchmarkFunc.*
		* For all benchmarks, compare current with devel: ./funcbench -v devel .* or ./funcbench -v devel
		* For BenchmarkFunc.*, compare current with 6d280 commit: ./funcbench -v 6d280 BenchmarkFunc.*
		* For BenchmarkFunc.*, compare between sub-benchmarks of same benchmark on current commit: ./funcbench -v . BenchmarkFunc.*
		* For BenchmarkFuncName, compare pr#35 with master: ./funcbench --nocomment --github-pr="35" master BenchmarkFuncName`,
	)

	// Options.
	app.HelpFlag.Short('h')
	app.Flag("verbose", "Verbose mode. Errors includes trace and commands output are logged.").
		Short('v').BoolVar(&cfg.verbose)
	app.Flag("nocomment", "Disable posting of comment using the GitHub API.").
		BoolVar(&cfg.nocomment)

	app.Flag("owner", "A Github owner or organisation name.").
		Default("prometheus").StringVar(&cfg.owner)
	app.Flag("repo", "This is the repository name.").
		Default("prometheus").StringVar(&cfg.repo)
	app.Flag("github-pr", "GitHub PR number to pull changes from and to post benchmark results.").
		IntVar(&cfg.ghPR)
	app.Flag("workspace", "Directory to clone GitHub PR.").
		Default("/tmp/funcbench").
		StringVar(&cfg.workspaceDir)
	app.Flag("result-cache", "Directory to store benchmark results.").
		Default("_dev/funcbench").
		StringVar(&cfg.resultsDir)

	app.Flag("bench-time", "Run enough iterations of each benchmark to take t, specified "+
		"as a time.Duration. The special syntax Nx means to run the benchmark N times").
		Short('t').Default("1s").DurationVar(&cfg.benchTime)
	app.Flag("timeout", "Benchmark timeout specified in time.Duration format, "+
		"disabled if set to 0. If a test binary runs longer than duration d, panic.").
		Short('d').Default("2h").DurationVar(&cfg.benchTimeout)

	app.Arg("target", "Can be one of '.', branch name or commit SHA of the branch "+
		"to compare against. If set to '.', branch/commit is the same as the current one; "+
		"funcbench will run once and try to compare between 2 sub-benchmarks. "+
		"Errors out if there are no sub-benchmarks.").
		Required().StringVar(&cfg.compareTarget)
	app.Arg("bench-func-regex", "Function regex to use for benchmark."+
		"Supports RE2 regexp and is fully anchored, by default will run all benchmarks.").
		Default(".*").
		StringVar(&cfg.benchFuncRegex) // TODO (geekodour) : validate regex?

	kingpin.MustParse(app.Parse(os.Args[1:]))
	logger := &logger{
		// Show file line with each log.
		Logger:  log.New(os.Stdout, "funcbech", log.Ltime|log.Lshortfile),
		verbose: cfg.verbose,
	}

	var g run.Group
	// Main routine.
	{
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			var (
				env Environment
				err error
			)

			// Setup Environment.
			e := environment{
				logger:        logger,
				benchFunc:     cfg.benchFuncRegex,
				compareTarget: cfg.compareTarget,
			}
			if cfg.ghPR == 0 {
				// Local Mode.
				env, err = newLocalEnv(e)
				if err != nil {
					return errors.Wrap(err, "environment create")
				}
			} else {
				// Github Mode.
				ghClient, err := newGitHubClient(ctx, cfg.owner, cfg.repo, cfg.ghPR, cfg.nocomment)
				if err != nil {
					return errors.Wrapf(err, "github client")
				}

				env, err = newGitHubEnv(ctx, e, ghClient, cfg.workspaceDir)
				if err != nil {
					if err := ghClient.postComment(fmt.Sprintf("%v. Could not setup environment, please check logs.", err)); err != nil {
						return errors.Wrap(err, "could not post error")
					}
					return errors.Wrap(err, "environment create")
				}
			}

			// ( ◔_◔)ﾉ Start benchmarking!
			cmps, err := startBenchmark(ctx, env, newBenchmarker(logger, env, &commander{verbose: cfg.verbose}, cfg.benchTime, cfg.benchTimeout, cfg.resultsDir))
			if err != nil {
				if pErr := env.PostErr(fmt.Sprintf("%v. Benchmark failed, please check logs.", err)); pErr != nil {
					return errors.Wrap(pErr, "could not log error")
				}
				return err
			}

			// Post results.
			// TODO (geekodour): probably post some kind of funcbench summary(?)
			return env.PostResults(cmps)

		}, func(err error) {
			cancel()
		})
	}
	// Listen for termination signals.
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return interrupt(logger, cancel)
		}, func(error) {
			close(cancel)
		})
	}

	if err := g.Run(); err != nil {
		logger.FatalError(errors.Wrap(err, "running command failed"))
	}
	logger.Println("exiting")
}

// startBenchmark returns the comparision results.
// 1. If target is same as current ref, run sub-benchmarks and return instead (TODO).
// 2. Execute benchmark against packages in the current worktree.
// 3. Cleanup of worktree in case funcbench was run previously and checkout target worktree.
// 4. Execute benchmark against packages in the new(target) worktree.
// 5. Return compared results.
func startBenchmark(
	ctx context.Context,
	env Environment,
	bench *Benchmarker,
) ([]BenchCmp, error) {

	wt, _ := env.Repo().Worktree()
	cmpWorkTreeDir := filepath.Join(wt.Filesystem.Root(), "_funcbench-cmp")

	ref, err := env.Repo().Head()
	if err != nil {
		return nil, errors.Wrap(err, "get head")
	}

	if _, err := bench.c.exec("sh", "-c", "git update-index -q --ignore-submodules --refresh && git diff-files --quiet --ignore-submodules --"); err != nil {
		return nil, errors.Wrap(err, "not clean worktree")
	}

	// Get info about target.
	targetCommit, compareWithItself, err := getTargetInfo(ctx, env.Repo(), env.CompareTarget())
	if err != nil {
		return nil, errors.Wrap(err, "getTargetInfo")
	}
	bench.logger.Println("Target:", targetCommit.String(), "Current Ref:", ref.Hash().String())

	if compareWithItself {
		bench.logger.Println("Assuming sub-benchmarks comparison.")
		subResult, err := bench.execBenchmark(wt.Filesystem.Root(), ref.Hash())
		if err != nil {
			return nil, errors.Wrap(err, "execute sub-benchmark")
		}

		cmps, err := bench.compareSubBenchmarks(subResult)
		if err != nil {
			return nil, errors.Wrap(err, "comparing sub benchmarks")
		}
		return cmps, nil
	}

	bench.logger.Println("Assuming comparing with target (clean workdir will be checked.)")

	// Execute benchmark A.
	newResult, err := bench.execBenchmark(wt.Filesystem.Root(), ref.Hash())
	if err != nil {
		return nil, errors.Wrapf(err, "execute benchmark for A: %v", ref.Name().String())
	}

	// Best effort cleanup and checkout new worktree.
	if err := os.RemoveAll(cmpWorkTreeDir); err != nil {
		return nil, errors.Wrapf(err, "delete worktree at %s", cmpWorkTreeDir)
	}

	// TODO (geekodour): switch to worktree remove once we decide not to support git<2.17
	if _, err := bench.c.exec("git", "worktree", "prune"); err != nil {
		return nil, errors.Wrap(err, "worktree prune")
	}

	bench.logger.Println("Checking out (in new workdir):", cmpWorkTreeDir, "commmit", targetCommit.String())
	if _, err := bench.c.exec("git", "worktree", "add", "-f", cmpWorkTreeDir, targetCommit.String()); err != nil {
		return nil, errors.Wrapf(err, "checkout %s in worktree %s", targetCommit.String(), cmpWorkTreeDir)
	}

	// Execute benchmark B.
	oldResult, err := bench.execBenchmark(cmpWorkTreeDir, targetCommit)
	if err != nil {
		return nil, errors.Wrapf(err, "execute benchmark for B: %v", env.CompareTarget())
	}

	// Compare B vs A.
	cmps, err := bench.compareBenchmarks(oldResult, newResult)
	if err != nil {
		return nil, errors.Wrap(err, "comparing benchmarks")
	}
	return cmps, nil
}

func interrupt(logger Logger, cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-c:
		logger.Println("caught signal", s, "Exiting.")
		return nil
	case <-cancel:
		return errors.New("canceled")
	}
}

// getTargetInfo returns the hash of the target,
// if target is the same as the current ref, set compareWithItself to true.
func getTargetInfo(ctx context.Context, repo *git.Repository, target string) (ref plumbing.Hash, compareWithItself bool, _ error) {
	if target == "." {
		return plumbing.Hash{}, true, nil
	}

	currRef, err := repo.Head()
	if err != nil {
		return plumbing.ZeroHash, false, err
	}

	if target == strings.TrimPrefix(currRef.Name().String(), "refs/heads/") || target == currRef.Hash().String() {
		return currRef.Hash(), true, errors.Errorf("target: %s is the same as current ref %s (or is on the same commit); No changes would be expected; Aborting", target, currRef.String())
	}

	commitHash := plumbing.NewHash(target)
	if !commitHash.IsZero() {
		return commitHash, false, nil
	}

	targetRef, err := repo.Reference(plumbing.NewBranchReferenceName(target), false)
	if err != nil {
		return plumbing.ZeroHash, false, err
	}

	return targetRef.Hash(), false, nil
}

type commander struct {
	verbose bool
}

func (c *commander) exec(command ...string) (string, error) {
	// TODO(bwplotka): Use context to kill command on interrupt.
	cmd := exec.Command(command[0], command[1:]...)
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b

	if c.verbose {
		// All to stdout.
		cmd.Stdout = io.MultiWriter(cmd.Stdout, os.Stdout)
		cmd.Stderr = io.MultiWriter(cmd.Stdout, os.Stdout)
	}
	if err := cmd.Run(); err != nil {
		out := b.String()
		if c.verbose {
			out = ""
		}
		return "", errors.Errorf("error: %v; Command out: %s", err, out)
	}

	return b.String(), nil
}
