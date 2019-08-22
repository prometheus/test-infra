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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
)

type githubClient struct {
	clt   *github.Client
	owner string
	repo  string
	pr    int
}

type commentMonitorClient struct {
	ghClient           githubClient
	allArgs            map[string]string
	inputFilePath      string
	outputDirPath      string
	regexString        string
	regex              *regexp.Regexp
	verifyUserDisabled bool
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	cmClient := commentMonitorClient{
		allArgs: make(map[string]string),
	}

	app := kingpin.New(filepath.Base(os.Args[0]), `commentMonitor github comment extract
	./commentMonitor -i /path/event.json -o /path "^myregex$"
	Example of comment template environment variable:
	COMMENT_TEMPLATE="The benchmark is starting. Your Github token is {{ index . "SOME_VAR" }}."`)
	app.HelpFlag.Short('h')
	app.Flag("input", "path to event.json").
		Short('i').
		Default("/github/workflow/event.json").
		StringVar(&cmClient.inputFilePath)
	app.Flag("output", "path to write args to").
		Short('o').
		Default("/github/home/commentMonitor").
		StringVar(&cmClient.outputDirPath)
	app.Flag("no-verify-user", "disable verifying user").
		BoolVar(&cmClient.verifyUserDisabled)
	app.Arg("regex", "Regex pattern to match").
		StringVar(&cmClient.regexString)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	err := os.MkdirAll(cmClient.outputDirPath, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	data, err := ioutil.ReadFile(cmClient.inputFilePath)
	if err != nil {
		log.Fatalln(err)
	}

	// Temporary fix for the new Github actions time format. This makes the time stamps unusable.
	reg := regexp.MustCompile("(.*)\"[0-9]+/[0-9]+/2019 [0-9]+:[0-9]+:[0-9]+ [AP]M(.*)")
	m := reg.FindSubmatch(data)
	if m != nil {
		txt := string(data)
		txt = reg.ReplaceAllString(txt, "$1\"2019-06-11T09:26:28Z$2")
		data = []byte(txt)
		log.Println("temp fix active")
	} else {
		log.Println(`WARNING: Github actions outputs correct date format so can
		remove the workaround fix in the code commentMonitor.
		https://github.com/google/go-github/issues/1254`)
	}
	// End of the temporary fix

	// Parsing event.json.
	event, err := github.ParseWebHook("issue_comment", data)
	if err != nil {
		log.Fatalln(err)
	}

	switch e := event.(type) {
	case *github.IssueCommentEvent:

		owner := *e.GetRepo().Owner.Login
		repo := *e.GetRepo().Name
		pr := *e.GetIssue().Number
		author := *e.Sender.Login
		authorAssociation := *e.GetComment().AuthorAssociation
		commentBody := *e.GetComment().Body

		// Setup commentMonitorClient.
		ctx := context.Background()
		cmClient.ghClient = newGithubClient(ctx, owner, repo, pr)

		// Validate comment if regexString provided.
		if cmClient.regexString != "" {
			cmClient.regex = regexp.MustCompile(cmClient.regexString)
			if !cmClient.regex.MatchString(commentBody) {
				log.Fatalf("matching command not found. comment validation failed")
			}
			log.Println("comment validation successful")
		}

		// Verify if user is allowed to perform activity.
		if !cmClient.verifyUserDisabled {
			var allowed bool
			allowedAssociations := []string{"COLLABORATOR", "MEMBER", "OWNER"}
			for _, a := range allowedAssociations {
				if a == authorAssociation {
					allowed = true
				}
			}
			if !allowed {
				log.Printf("author is not a member or collaborator")
				b := fmt.Sprintf("@%s is not a org member nor a collaborator and cannot execute benchmarks.", author)
				if err := cmClient.ghClient.postComment(ctx, b); err != nil {
					log.Fatalf("%v : couldn't post comment", err)
				}
				os.Exit(1)
			}
			log.Println("author is a member or collaborator")
		}

		// Extract args if regexString provided.
		if cmClient.regexString != "" {
			// Add comment arguments.
			commentArgs := cmClient.regex.FindStringSubmatch(commentBody)[1:]
			commentArgsNames := cmClient.regex.SubexpNames()[1:]
			for i, argName := range commentArgsNames {
				if argName == "" {
					log.Fatalln("using named groups is mandatory")
				}
				cmClient.allArgs[argName] = commentArgs[i]
			}

			// Add non-comment arguments if any.
			cmClient.allArgs["PR_NUMBER"] = strconv.Itoa(pr)

			err := cmClient.writeArgs()
			if err != nil {
				log.Fatalf("%v: could not write args to fs", err)
			}
		}

		// Post generated comment to Github pr if COMMENT_TEMPLATE is set.
		if os.Getenv("COMMENT_TEMPLATE") != "" {
			err := cmClient.generateAndPostComment(ctx)
			if err != nil {
				log.Fatalf("%v: could not post comment", err)
			}
		}

		// Set label to Github pr if LABEL_NAME is set.
		if os.Getenv("LABEL_NAME") != "" {
			if err := cmClient.ghClient.createLabel(ctx, os.Getenv("LABEL_NAME")); err != nil {
				log.Fatalf("%v : couldn't set label", err)
			}
			log.Println("label successfully set")
		}

	default:
		log.Fatalln("only issue_comment event is supported")
	}
}

func (c commentMonitorClient) writeArgs() error {
	for filename, content := range c.allArgs {
		data := []byte(content)
		err := ioutil.WriteFile(filepath.Join(c.outputDirPath, filename), data, 0644)
		if err != nil {
			return fmt.Errorf("%v: could not write arg to filesystem", err)
		}
		log.Printf("file added: %v", filepath.Join(c.outputDirPath, filename))
	}
	return nil
}

func (c commentMonitorClient) generateAndPostComment(ctx context.Context) error {
	// Add all env vars to allArgs.
	for _, e := range os.Environ() {
		tmp := strings.Split(e, "=")
		c.allArgs[tmp[0]] = tmp[1]
	}
	// Generate the comment template.
	var buf bytes.Buffer
	commentTemplate := template.Must(template.New("Comment").
		Parse(os.Getenv("COMMENT_TEMPLATE")))
	if err := commentTemplate.Execute(&buf, c.allArgs); err != nil {
		return err
	}
	// Post the comment.
	if err := c.ghClient.postComment(ctx, buf.String()); err != nil {
		return fmt.Errorf("%v : couldn't post generated comment", err)
	}
	log.Println("comment successfully posted")
	return nil
}

// githubClient Methods
func newGithubClient(ctx context.Context, owner, repo string, pr int) githubClient {
	ghToken := os.Getenv("GITHUB_TOKEN")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	return githubClient{
		clt:   github.NewClient(tc),
		owner: owner,
		repo:  repo,
		pr:    pr,
	}
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
