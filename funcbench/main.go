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
	app := kingpin.New(
		filepath.Base(os.Args[0]),
		"Benchmark and compare your Go code between sub benchmarks or commits.",
	)

	// Options.
	app.HelpFlag.Short('h')
	verbose := app.Flag("verbose", "Verbose mode. Errors includes trace and commands output are logged.").Short('v').Bool()

	// TODO(bwplotka): Why not just passing full import path? Easier (:
	owner := app.Flag("owner", "A Github owner or organisation name.").Default("prometheus").Short('o').String()
	repo := app.Flag("repo", "This is the repository name.").Default("prometheus").Short('r').String()
	// TODO(bwplotka): Should we run in worktree for consistency?
	gitHubPRNumber := app.Flag("github-pr", "GitHub Pull Request number (#<num>) that should used to pull the latest changes"+
		"from and used for posting comments. NOTE: **This has to be run from the GithubAction.** If none provided, local mode is enabled.").
		Short('p').Int()

	compareTarget := app.Arg("branch/commit/<.>", "Branch, commit SHA of the branch to compare benchmarks against."+
		"If `.` of branch/commit is the same as the current one, funcbench will run once and try to compare between 2 sub-benchmarks.").String()
	benchFunc := app.Arg("func regexp/<.>", "Function to use for benchmark. Supports RE2 regexp or `.` to run all benchmarks.").String()

	benchTime := app.Flag("bench-time", `Run enough iterations of each benchmark to take t, specified
as a time.Duration (for example, -benchtime 1h30s).
The default is 1 second (1s).
The special syntax Nx means to run the benchmark N times
(for example, -benchtime 100x).`).Short('t').Default("1s").Duration()
	benchTimeout := app.Flag("timeout", `If each test binary runs longer than duration d, panic.
If d is 0, the timeout is disabled.
The default is 2 hours (2h).`).Default("2h").Duration()
	resultCacheDir := app.Flag("result-cache", "Directory to store output for given func name and commit sha. Useful for local runs.").String()
	workspace := app.Flag("workspace", "A directory where source code will be cloned to.").Default(os.Getenv("WORKSPACE")).Short('w').String()

	kingpin.MustParse(app.Parse(os.Args[1:]))
	logger := &logger{
		// Show file line with each log.
		Logger:  log.New(os.Stdout, "funcbech", log.Ltime|log.Lshortfile),
		verbose: *verbose,
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

			if *gitHubPRNumber == 0 {
				env, err = newLocalEnv(environment{
					logger:        logger,
					benchFunc:     *benchFunc,
					compareTarget: *compareTarget,
					home:          os.Getenv("HOME"),
				})
				if err != nil {
					return errors.Wrap(err, "new env")
				}
				logger.Printf("funcbench start [Local Mode]: Benchmarking current version versus %q for benchmark funcs: %q\n", *compareTarget, *benchFunc)
			} else {
				if *owner == "" || *repo == "" {
					return errors.New("funcbench in GitHub Mode requires --owner and --repo flags to be specified")
				}
				env, err = newGitHubEnv(ctx, environment{
					logger:        logger,
					benchFunc:     *benchFunc,
					compareTarget: *compareTarget,
					home:          *workspace,
				}, *owner, *repo, *gitHubPRNumber)
				if err != nil {
					return errors.Wrap(err, "new env")
				}
				logger.Printf("funcbench start [GitHub Mode]: Benchmarking %q (PR-%d) versus %q for benchmark funcs: %q\n", fmt.Sprintf("%s/%s", *owner, *repo), *gitHubPRNumber, *compareTarget, *benchFunc)
			}

			// ( ◔_◔)ﾉ Start benchmarking!
			return funcbench(ctx, env, newBenchmarker(logger, env, &commander{verbose: *verbose}, *benchTime, *benchTimeout, *resultCacheDir))

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

func funcbench(
	ctx context.Context,
	env Environment,
	bench *Benchmarker,
) error {
	wt, _ := env.Repo().Worktree()
	ref, err := env.Repo().Head()
	if err != nil {
		return errors.Wrap(err, "get head")
	}

	if _, err := bench.c.exec("bash", "-c", "git update-index -q --ignore-submodules --refresh && git diff-files --quiet --ignore-submodules --"); err != nil {
		return errors.Wrap(err, "not clean worktree")
	}

	// 1. Execute benchmark against packages in the current directory.
	newResult, err := bench.execBenchmark(wt.Filesystem.Root(), ref.Hash())
	if err != nil {
		// TODO(bwplotka): Just defer posting all errors?
		if pErr := env.PostErr("Go bench test for this pull request failed"); pErr != nil {
			return errors.Errorf("error: %v occured while processing error: %v", pErr, err)
		}
		return errors.Wrap(err, "exec benchmark A")
	}

	var oldResult string

	targetCommit, compareWithItself, err := compareTargetRef(ctx, env.Repo(), env.CompareTarget())
	if err != nil {
		return errors.Wrap(err, "compareTargetRef")
	}

	if compareWithItself {
		bench.logger.Println("Target:", env.CompareTarget(), "is `.`; Assuming sub-benchmarks comparison.")

		// 2a. Compare sub benchmarks. TODO.
		cmps, err := bench.compareSubBenchmarks(newResult)
		if err != nil {
			if pErr := env.PostErr("`benchcmp` failed."); pErr != nil {
				return errors.Errorf("error: %v occured while processing error: %v", pErr, err)
			}
			return errors.Wrap(err, "compare sub benchmarks")
		}
		return errors.Wrap(env.PostResults(cmps), "post results")
	}

	bench.logger.Println("Target:", env.CompareTarget(), "is evaluated to be ", targetCommit.String(), ". Assuming comparing with this one (clean workdir will be checked.)")
	// 2b. Compare with target commit/branch.

	// 3. Best effort cleanup of worktree.
	cmpWorkTreeDir := filepath.Join(wt.Filesystem.Root(), "_funcbench-cmp")

	_, _ = bench.c.exec("git", "worktree", "remove", cmpWorkTreeDir)
	bench.logger.Println("Checking out (in new workdir):", cmpWorkTreeDir, "commmit", ref.String())
	if _, err := bench.c.exec("git", "worktree", "add", "-f", cmpWorkTreeDir, ref.Hash().String()); err != nil {
		return errors.Wrapf(err, "failed to checkout %s in worktree %s", ref.String(), cmpWorkTreeDir)
	}

	// 4. Benchmark in new worktree.
	oldResult, err = bench.execBenchmark(cmpWorkTreeDir, targetCommit)
	if err != nil {
		if pErr := env.PostErr(fmt.Sprintf("Go bench test for target %s failed", env.CompareTarget())); pErr != nil {
			return errors.Errorf("error: %v occured while processing error: %v", pErr, err)
		}
		return errors.Wrap(err, "exec bench B")
	}

	// 5. Compare old vs new.
	cmps, err := bench.compareBenchmarks(oldResult, newResult)
	if err != nil {
		if pErr := env.PostErr("`benchcmp` failed."); pErr != nil {
			return errors.Errorf("error: %v occured while processing error: %v", pErr, err)
		}
		return errors.Wrap(err, "compare benchmarks")
	}
	return errors.Wrap(env.PostResults(cmps), "post results")
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

func compareTargetRef(ctx context.Context, repo *git.Repository, target string) (ref plumbing.Hash, compareWithItself bool, _ error) {
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

	if err := repo.FetchContext(ctx, &git.FetchOptions{}); err != nil && err != git.NoErrAlreadyUpToDate {
		return plumbing.ZeroHash, false, err
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
