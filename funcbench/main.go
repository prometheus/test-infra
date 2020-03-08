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
	comment          bool
	client           *github.Client
}

type benchmarkTester struct {
	bechRegex string
	home      string
}

func main() {
	// Show file line with each log.
	log.SetFlags(log.Ltime | log.Lshortfile)
	app := kingpin.New(
		filepath.Base(os.Args[0]),
		"benchmark result posting and formating tool.",
	)
	app.HelpFlag.Short('h')

	owner := app.Flag("owner", "It's a Github owner or organisation name.").Default("prometheus").Short('o').String()
	repo := app.Flag("repo", "This is the repository name.").Default("prometheus").Short('r').String()

	prNumber := app.Flag("pullrequest", "It's a new pull request ID, the benchmark will be run using a copy of its code.").Short('p').Required().Int()
	branch := app.Arg("branch", "A name of branch, will be used as a baseline in benchmark.").Required().String()
	regex := app.Arg("regex", "A regular expression will be used in '-bench regexp' during 'go test', check 'go help testflag' for details.").Default(".").String()

	postComment := app.Flag("comment", "If set to false, it will not post the benchmark result back to Github. This is mainly for local testing.").Default("true").Bool()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	log.Printf(
		"Start testing %s/%s PR-%d VS branch %s, on benchmark `%s`. With comment: %v.\n",
		*owner, *repo, *prNumber, *branch, *regex, *postComment,
	)

	var (
		ghClient  *gitHubClient
		gitClient *gitClient
	)

	ghClient = &gitHubClient{
		client:           newClient(os.Getenv("GITHUB_TOKEN")),
		owner:            *owner,
		repo:             *repo,
		prNumber:         *prNumber,
		comment:          *postComment,
		latestCommitHash: os.Getenv("GITHUB_SHA"),
	}

	gitClient, err := newGitClient(*owner, *repo, *branch)
	if err != nil {
		log.Fatalln(err)
	}

	benchTest, err := newBenchmarkTester(*regex)
	if err != nil {
		log.Fatalln(err)
	}
	logLink := fmt.Sprintf("Full logs at: https://github.com/%s/%s/commit/%s/checks", ghClient.owner, ghClient.repo, ghClient.latestCommitHash)

	if err := gitClient.cloneRepository(); err != nil {
		log.Fatalln(err)
	}

	if err := gitClient.checkoutPR(*prNumber); err != nil {
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

func (c *gitHubClient) postComment(comment string) error {
	if c.comment {
		issueComment := &github.IssueComment{Body: github.String(comment)}
		_, _, err := c.client.Issues.CreateComment(context.Background(), c.owner, c.repo, c.prNumber, issueComment)
		return err
	}
	return nil
}

func newGitClient(owner, repo, branch string) (*gitClient, error) {
	_, _, err := execCommand("git", "config", "--global", "user.email", "prombench@example.com")
	if err != nil {
		return nil, err
	}
	_, _, err = execCommand("git", "config", "--global", "user.name", "Prombench Bot Junior")
	if err != nil {
		return nil, err
	}

	c := gitClient{
		branch: branch,
		owner:  owner,
		repo:   repo,
	}

	return &c, err
}

func (c *gitClient) cloneRepository() error {
	if err := os.Chdir(os.Getenv("GITHUB_WORKSPACE")); err != nil {
		return err
	}
	_, _, err := execCommand("git", "clone", fmt.Sprintf("https://github.com/%s/%s.git", c.owner, c.repo))
	if err != nil {
		return err
	}
	return nil
}

// checkoutToPullRequest applies changes from the pull request to the working tree
// of the branch that is being compared.
func (c *gitClient) checkoutPR(num int) error {
	if err := os.Chdir(filepath.Join(os.Getenv("GITHUB_WORKSPACE"), c.repo)); err != nil {
		return errors.Wrap(err, "changing to GITHUB_WORKSPACE dir")
	}
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
	return filepath.Walk(filepath.Join(os.Getenv("GITHUB_WORKSPACE"), c.repo), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go") {
			_, _, err := execCommand("git", "checkout", "--", path)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func newBenchmarkTester(benchRegex string) (*benchmarkTester, error) {
	if err := os.Setenv("GO111MODULE", "on"); err != nil {
		return nil, err
	}

	if err := os.Setenv("CGO_ENABLED", "0"); err != nil {
		return nil, err
	}

	bench := benchmarkTester{
		home:      os.Getenv("HOME"),
		bechRegex: benchRegex,
	}

	_, _, err := execCommand("go", "get", "golang.org/x/tools/cmd/benchcmp")

	return &bench, err
}

func (bench *benchmarkTester) compareBenchmarks(old, new string) (string, error) {
	out, _, err := execCommand(filepath.Join(os.Getenv("GOPATH"), "/bin/benchcmp"), "-mag", old, new)
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
	out, exitCode, err = execCommand("go", "test", "-bench", fmt.Sprintf("^%s$", bench.bechRegex), "-benchmem", "-v", "./...")
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
