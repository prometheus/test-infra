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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
)

type gitClient struct {
	owner  string
	branch string
	repo   string
}
type gitHubClient struct {
	owner            string
	repo             string
	latestCommitHash string
	prNumber         int
	client           *github.Client
}

type benchmarkTester struct {
	bechRegex       string
	home            string
	raceFlagEnabled bool
}

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "benchmark result posting and formating tool.\n-i location of github hook file (even.json)")
	app.HelpFlag.Short('h')
	input := app.Flag("input", "path to event.json").Short('i').Default("/github/workflow/event.json").String()
	kingpin.MustParse(app.Parse(os.Args[1:]))

	data, err := ioutil.ReadFile(*input)
	if err != nil {
		log.Fatalln(err)
	}

	event, err := github.ParseWebHook("issue_comment", data)
	if err != nil {
		log.Fatalln(err)
	}
	var (
		ghClient  *gitHubClient
		gitClient *gitClient
		prNumber  int
	)
	switch eventType := event.(type) {
	case *github.IssueCommentEvent:
		ghClient = newGitHubClient(eventType)
		gitClient, err = newGitClient(eventType)
		if err != nil {
			log.Fatalln(err)
		}
		prNumber = *eventType.GetIssue().Number

	default:
		log.Fatalln("only issue_comment event is supported")
	}
	benchTest, err := newBenchmarkTester()
	if err != nil {
		log.Fatalln(err)
	}
	logLink := fmt.Sprintf("Full logs at: https://github.com/%s/%s/commit/%s/checks", ghClient.owner, ghClient.repo, ghClient.latestCommitHash)

	if err := os.Chdir(os.Getenv("GITHUB_WORKSPACE")); err != nil {
		log.Fatalln(err)
	}

	if err := gitClient.cloneRepository(); err != nil {
		log.Fatalln(err)
	}

	if err := gitClient.checkoutPR(prNumber); err != nil {
		comment := fmt.Sprintf("Switch to a pull request branch failed. %s", logLink)
		if postCommentErr := ghClient.postComment(comment); postCommentErr != nil {
			log.Fatalf("Error posting a comment for `checkoutToPullRequest` command execution error. checkoutToPullRequest err:%v, postComment err:%v", err, postCommentErr)
		}
		log.Fatalln(err)
	}

	prResults, err := benchTest.execBenchmark()
	if err != nil {
		comment := fmt.Sprintf("Go bench test for this pull request failed. %s", logLink)
		if postCommentErr := ghClient.postComment(comment); postCommentErr != nil {
			log.Fatalf("An error: %v occured while processing error: %v", postCommentErr, err)
		}
		log.Fatalln(err)
	}

	if err := gitClient.revertPRChanges(); err != nil {
		log.Fatalln(err)
	}

	branchResults, err := benchTest.execBenchmark()
	if err != nil {
		comment := fmt.Sprintf("Go bench test for this pull request failed. %s", logLink)
		if postCommentErr := ghClient.postComment(comment); postCommentErr != nil {
			log.Fatalf("An error: %v occured while processing error: %v", postCommentErr, err)
		}
		log.Fatalln(err)
	}

	comparisonTable, err := benchTest.compareBenchmarks(branchResults, prResults)
	if err != nil {
		comment := fmt.Sprintf("Error: `benchcmp` failed. %s", logLink)
		if postCommentErr := ghClient.postComment(comment); postCommentErr != nil {
			log.Fatalf("An error: %v occured while processing error: %v", postCommentErr, err)
		}
		log.Fatalln(err)
	}
	if err := ghClient.postComment(comparisonTable); err != nil {
		log.Fatalln(err)
	}

}

func newGitHubClient(event *github.IssueCommentEvent) *gitHubClient {
	c := gitHubClient{
		client:           newClient(os.Getenv("GITHUB_TOKEN")),
		owner:            *event.GetRepo().Owner.Login,
		repo:             *event.GetRepo().Name,
		prNumber:         *event.GetIssue().Number,
		latestCommitHash: os.Getenv("GITHUB_SHA"),
	}
	log.Printf("GH Client: %+v\n", c)
	return &c
}

func (c *gitHubClient) postComment(comment string) error {
	issueComment := &github.IssueComment{Body: github.String(comment)}
	_, _, err := c.client.Issues.CreateComment(context.Background(), c.owner, c.repo, c.prNumber, issueComment)
	return err
}

func newGitClient(event *github.IssueCommentEvent) (*gitClient, error) {
	_, _, err := execCommand("git", "config", "--global", "user.email", "prombench@example.com")
	if err != nil {
		return nil, err
	}
	_, _, err = execCommand("git", "config", "--global", "user.name", "Prombench Bot Junior")
	if err != nil {
		return nil, err
	}

	branch, err := ioutil.ReadFile("/github/home/commentMonitor/BRANCH")
	c := gitClient{
		branch: string(branch),
		owner:  *event.GetRepo().Owner.Login,
		repo:   *event.GetRepo().Name,
	}

	log.Printf("Git Client: %+v\n", c)
	return &c, err
}

func (c *gitClient) cloneRepository() error {
	_, _, err := execCommand("git", "clone", fmt.Sprintf("https://github.com/%s/%s.git", c.owner, c.repo))
	if err != nil {
		return err
	}
	err = os.Chdir(c.repo)
	return err
}

// checkoutToPullRequest applies changes from the pull request to the working tree
// of the branch that is being compared.
func (c *gitClient) checkoutPR(num int) error {
	_, _, err := execCommand("git", "fetch", "origin", fmt.Sprintf("pull/%d/head:pullrequest", num))
	if err != nil {
		return err
	}
	_, _, err = execCommand("git", "checkout", c.branch)
	if err != nil {
		return err
	}
	_, exitCode, err := execCommand("git", "merge", "--squash", "--no-commit", "pullrequest")
	if err != nil || exitCode != 0 {
		return errors.Wrap(err, "Pull request merge failed.")
	}
	_, _, err = execCommand("git", "reset")
	return err
}

func (c *gitClient) revertPRChanges() error {
	return filepath.Walk(os.Getenv("GITHUB_WORKSPACE"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || strings.Contains(path, ".git/") || strings.HasSuffix(info.Name(), "test.go") {
			return nil
		}

		data, _, err := execCommand("git", "checkout", "--", path)
		if err != nil {
			log.Println("Following error occured during checkout: ", data, "If the error is caused by an untracted file, it will be deleted.")

			// New file appeared and is not tracked, then it can be removed.
			if strings.Contains(data, "did not match any file") {
				if err := os.Remove(path); err != nil {
					return errors.Errorf("Error: %v; Command out: %s", err, string(data))
				}
				return nil
			}
			return err
		}
		return nil
	})
}

func newBenchmarkTester() (*benchmarkTester, error) {
	if err := os.Setenv("GO111MODULE", "on"); err != nil {
		return nil, err
	}

	benchRegex, err := ioutil.ReadFile("/github/home/commentMonitor/REGEX")
	if err != nil {
		return nil, err
	}
	raceArgument, err := ioutil.ReadFile("/github/home/commentMonitor/RACE")
	if err != nil {
		return nil, err
	}

	bench := benchmarkTester{
		raceFlagEnabled: string(raceArgument) != "-no-race",
		home:            os.Getenv("HOME"),
		bechRegex:       string(benchRegex),
	}

	_, _, err = execCommand("go", "get", "golang.org/x/tools/cmd/benchcmp")

	log.Printf("Benchmark Tester: %+v\n", bench)
	return &bench, err
}

func (bench *benchmarkTester) compareBenchmarks(old, new string) (string, error) {
	out, _, err := execCommand(strings.Join([]string{os.Getenv("GOPATH"), "/bin/benchcmp"}, ""), "-mag", old, new)
	log.Println("Benchmark comparision output: ", out)

	if strings.Count(out, "\n") < 2 {
		return out, errors.New("error: `go test` did not match any `BenchmarkXxx` functions")
	}
	return formatComment(out), err
}

func (bench *benchmarkTester) execBenchmark() (string, error) {
	var (
		out      string
		exitCode int
		err      error
	)
	if bench.raceFlagEnabled {
		out, exitCode, err = execCommand("go", "test", "-bench", fmt.Sprintf("^%s$", bench.bechRegex), "-benchmem", "-v", "./...")
	} else {
		out, exitCode, err = execCommand("go", "test", "-bench", fmt.Sprintf("^%s$", bench.bechRegex), "-benchmem", "-race", "-v", "./...")
	}
	log.Println("Executing benchmark with REGEX ", bench.bechRegex,
		"Benchmark output: ", out)
	if err != nil || exitCode != 0 {
		return "", errors.Wrap(err, "Benchmark ended with an error.")
	}

	tempFile, err := ioutil.TempFile("", "benchmark")
	if err != nil {
		return "", err
	}

	if _, err := tempFile.Write([]byte(out)); err != nil {
		return "", err
	}
	err = tempFile.Close()
	return tempFile.Name(), err
}

func newClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	clt := github.NewClient(tc)
	return clt
}

func formatComment(rawTable string) string {
	tableContent := strings.Split(rawTable, "\n")
	for i := 0; i <= len(tableContent)-1; i++ {
		e := tableContent[i]
		switch {
		case e == "":

		case strings.Contains(e, "old ns/op"):
			e = "| Benchmark | Old ns/op | New ns/op | Delta |"
			tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

		case strings.Contains(e, "old MB/s"):
			e = "| Benchmark | Old MB/s | New MB/s | Speedup |"
			tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

		case strings.Contains(e, "old allocs"):
			e = "| Benchmark | Old allocs | New allocs | Delta |"
			tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

		case strings.Contains(e, "old bytes"):
			e = "| Benchmark | Old bytes | New bytes | Delta |"
			tableContent = append(tableContent[:i+1], append([]string{"|-|-|-|-|"}, tableContent[i+1:]...)...)

		default:
			// Replace spaces with "|".
			e = strings.Join(strings.Fields(e), "|")
		}
		tableContent[i] = e
	}
	return strings.Join(tableContent, "\n")

}

func execCommand(command ...string) (string, int, error) {
	cmd := exec.Command(command[0], command[1:]...)
	data, err := cmd.CombinedOutput()
	if err != nil {
		return "", 1, errors.Errorf("Error: %v; Command out: %s", err, string(data))
	}
	return string(data), cmd.ProcessState.ExitCode(), nil
}
