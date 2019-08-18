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
	clt      *github.Client
	owner    string
	repo     string
	prnumber int
}

type commentMonitorClient struct {
	ghClient                githubClient
	commentTemplateVars     map[string]string
	labelName               string
	inputFilePath           string
	outputDirPath           string
	regexString             string
	regex                   *regexp.Regexp
	namedArgs               []string
	verifyUserDisabled      bool
	validateCommentDisabled bool
	extractArgsDisabled     bool
	postCommentDisabled     bool
	labelSetDisabled        bool
}

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	cmClient := commentMonitorClient{}

	app := kingpin.New(filepath.Base(os.Args[0]), `commentMonitor github comment extract
	./commentMonitor -i /path/event.json -o /path "^myregex$"
	Example of comment template environment variable:
	COMMENT_TEMPLATE="The benchmark is starting. Your Github token is {{ index . "GITHUB_TOKEN" }}."`)
	app.HelpFlag.Short('h')

	app.Flag("input", "path to event.json").
		Short('i').
		Default("/github/workflow/event.json").
		StringVar(&cmClient.inputFilePath)
	app.Flag("output", "path to write args to").
		Short('o').
		Default("/github/home/commentMonitor").
		StringVar(&cmClient.outputDirPath)
	app.Flag("named-arg", "Should be in the format: --named-arg=ARG_NAME:<regex_for_arg>").
		StringsVar(&cmClient.namedArgs)
	app.Flag("label-name", "Name of the label to set").
		Default("prombench").
		StringVar(&cmClient.labelName)

	// Disable flags.
	app.Flag("no-verify-user", "disable verifying user").
		BoolVar(&cmClient.verifyUserDisabled)
	app.Flag("no-comment-validate", "disable comment validation").
		BoolVar(&cmClient.validateCommentDisabled)
	app.Flag("no-args-extract", "disable args extraction").
		BoolVar(&cmClient.extractArgsDisabled)
	app.Flag("no-post-comment", "disable comment posting").
		BoolVar(&cmClient.postCommentDisabled)
	app.Flag("no-label-set", "disable setting label").
		BoolVar(&cmClient.labelSetDisabled)

	app.Arg("regex", "Regex pattern to match").
		StringVar(&cmClient.regexString)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	// Reading event.json.
	err := os.MkdirAll(cmClient.outputDirPath, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	data, err := ioutil.ReadFile(cmClient.inputFilePath)
	if err != nil {
		log.Fatalln(err)
	}
	// Parsing event.json.
	event, err := github.ParseWebHook("issue_comment", data)
	if err != nil {
		log.Fatalln(err)
	}

	switch e := event.(type) {
	case *github.IssueCommentEvent:

		owner := *e.GetRepo().Owner.Login
		repo := *e.GetRepo().Name
		prnumber := *e.GetIssue().Number
		author := *e.Sender.Login
		authorAssociation := *e.GetComment().AuthorAssociation
		commentBody := *e.GetComment().Body

		// Setup commentMonitorClient.
		ctx := context.Background()
		cmClient.ghClient = newGithubClient(ctx, owner, repo, prnumber)
		cmClient.commentTemplateVars = make(map[string]string) // initialization required

		// Validate comment.
		if !cmClient.validateCommentDisabled {
			if cmClient.regexString == "" {
				log.Fatalln(`comment validation is enabled but no regex to validate against,
				please use --no-comment-validate flag to disable comment validation`)
			}
			cmClient.regex = regexp.MustCompile(cmClient.regexString)
			if !cmClient.regex.MatchString(commentBody) {
				log.Fatalf("matching command not found. comment validation failed")
			}
			log.Println("comment validation successful")
		}

		// Verify user.
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

		// Extract args, write them to fs, put them in commentTemplateVars.
		if !cmClient.extractArgsDisabled {
			if cmClient.validateCommentDisabled {
				log.Fatalln(`comment validation must be enabled to use args extraction,
				please use --no-args-extract flag to disable extracting arguments`)
			}
			args := cmClient.regex.FindStringSubmatch(commentBody)
			args = append(args, strconv.Itoa(prnumber))
			err := cmClient.extractArgs(args)
			if err != nil {
				log.Fatalf("%v: could not extract args", err)
			}
		}

		// Post generated comment to Github pr.
		if !cmClient.postCommentDisabled {
			if os.Getenv("COMMENT_TEMPLATE") == "" {
				log.Fatalln(`COMMENT_TEMPLATE env var empty,
				please use --no-post-comment flag to disable commenting`)
			}
			// Add all env vars to commentTemplateVars.
			for _, e := range os.Environ() {
				tmp := strings.Split(e, "=")
				cmClient.commentTemplateVars[tmp[0]] = tmp[1]
			}
			// Generate the comment template.
			var buf bytes.Buffer
			commentTemplate := template.Must(template.New("Comment").
				Parse(os.Getenv("COMMENT_TEMPLATE")))
			if err := commentTemplate.Execute(&buf, cmClient.commentTemplateVars); err != nil {
				log.Fatalln(err)
			}
			// Post the comment.
			if err := cmClient.ghClient.postComment(ctx, buf.String()); err != nil {
				log.Fatalf("%v : couldn't post generated comment", err)
			}
		}

		// Set label to Github pr.
		if !cmClient.labelSetDisabled {
			if cmClient.labelName == "" {
				log.Fatalln(`--label-name must set when setting a label,
				please use --no-label-set flag to disable setting label`)
			}
			if err := cmClient.ghClient.createLabel(ctx, cmClient.labelName); err != nil {
				log.Fatalf("%v : couldn't set label", err)
			}
		}

	default:
		log.Fatalln("commentMonitor only supports issue_comment event")
	}
}

func (c commentMonitorClient) extractArgs(args []string) error {
	namedArgsMap := make(map[string]regexp.Regexp)
	for _, arg := range c.namedArgs {
		re := regexp.MustCompile(`^\w+:.+$`)
		if !re.MatchString(arg) {
			return fmt.Errorf("named arg should be of the format: --named-arg=ARG_NAME:<regex_for_arg>")
		}
		namedArg := strings.Split(arg, ":")
		namedArgRegex := regexp.MustCompile(namedArg[1])
		namedArgsMap[namedArg[0]] = *namedArgRegex
	}

	for i, arg := range args[1:] {
		var filename string
		for name, re := range namedArgsMap {
			if re.MatchString(arg) {
				filename = name
			}
		}

		if filename == "" {
			filename = fmt.Sprintf("ARG_%v", i)
		}

		data := []byte(arg)
		err := ioutil.WriteFile(filepath.Join(c.outputDirPath, filename), data, 0644)
		if err != nil {
			return fmt.Errorf("%v: could not write arg to filesystem", err)
		}

		// Add argument to commentTemplateVars
		c.commentTemplateVars[filename] = arg

		log.Printf("file added: %v", filepath.Join(c.outputDirPath, filename))
	}

	return nil
}

// githubClient Methods
func newGithubClient(ctx context.Context, owner, repo string, prnumber int) githubClient {
	ghToken := os.Getenv("GITHUB_TOKEN")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	return githubClient{
		clt:      github.NewClient(tc),
		owner:    owner,
		repo:     repo,
		prnumber: prnumber,
	}
}

func (c githubClient) postComment(ctx context.Context, commentBody string) error {
	issueComment := &github.IssueComment{Body: github.String(commentBody)}
	_, _, err := c.clt.Issues.CreateComment(ctx, c.owner, c.repo, c.prnumber, issueComment)
	return err
}

func (c githubClient) createLabel(ctx context.Context, labelName string) error {
	benchmarkLabel := []string{labelName}
	_, _, err := c.clt.Issues.AddLabelsToIssue(ctx, c.owner, c.repo, c.prnumber, benchmarkLabel)
	return err
}
