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
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v3"
)

type commentMonitorConfig struct {
	verifyUserDisabled bool
	webhook            bool
	eventMapFilePath   string
	eventMap           webhookEventMaps
}

// Structure of eventmap.yaml file.
type webhookEventMap struct {
	EventType       string `yaml:"event_type"`
	CommentTemplate string `yaml:"comment_template"`
	RegexString     string `yaml:"regex_string"`
}

type webhookEventMaps []webhookEventMap

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	cmConfig := commentMonitorConfig{}

	app := kingpin.New(filepath.Base(os.Args[0]), `commentMonitor GithubAction - Post and monitor GitHub comments.`)
	app.HelpFlag.Short('h')
	app.Flag("no-verify-user", "disable verifying user").
		BoolVar(&cmConfig.verifyUserDisabled)
	app.Flag("webhook", "enable webhook mode").
		BoolVar(&cmConfig.webhook)
	app.Flag("eventmap", "Filepath to eventmap file.").
		Default("./eventmap.yml").
		StringVar(&cmConfig.eventMapFilePath)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if cmConfig.webhook {
		// Get eventmap file.
		data, err := ioutil.ReadFile(cmConfig.eventMapFilePath)
		if err != nil {
			log.Fatalln(err)
		}
		// TODO: Do strict checking.
		err = yaml.Unmarshal(data, &cmConfig.eventMap)
		if err != nil {
			log.Fatalf("cannot unmarshal data: %v", err)
		}
		if len(cmConfig.eventMap) == 0 {
			log.Fatalln("eventmap empty")
		}
		http.HandleFunc("/", cmConfig.webhookExtract)
		log.Fatal(http.ListenAndServe(":8080", nil))
	}

	// If not run as webhook, just post comment from COMMENT_TEMPLATE.
	cmConfig.postComment()

}

func newGithubClientForIssueComments(ctx context.Context, e *github.IssueCommentEvent) (*githubClient, error) {
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		return nil, fmt.Errorf("env var missing")
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	return &githubClient{
		clt:               github.NewClient(tc),
		owner:             *e.GetRepo().Owner.Login,
		repo:              *e.GetRepo().Name,
		pr:                *e.GetIssue().Number,
		author:            *e.Sender.Login,
		authorAssociation: *e.GetComment().AuthorAssociation,
		commentBody:       *e.GetComment().Body,
	}, nil
}

func newGithubClient(ctx context.Context, owner, repo string, pr int) (*githubClient, error) {
	ghToken := os.Getenv("GITHUB_TOKEN")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	return &githubClient{
		clt:   github.NewClient(tc),
		owner: owner,
		repo:  repo,
		pr:    pr,
	}, nil
}

func (c commentMonitorConfig) webhookExtract(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unable to read webhook body", http.StatusBadRequest)
	}

	// Setup commentMonitor client.
	cmClient := commentMonitorClient{
		allArgs:  make(map[string]string),
		eventMap: c.eventMap,
	}

	// Parse webhook event.
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		http.Error(w, "unable to parse webhook", http.StatusBadRequest)
	}

	switch e := event.(type) {
	case *github.IssueCommentEvent:

		// Setup github client.
		ctx := context.Background()
		cmClient.ghClient, err = newGithubClientForIssueComments(ctx, e)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not create GitHub client", http.StatusBadRequest)
			return
		}

		// Validate regex.
		err := cmClient.validateRegex()
		if err != nil {
			log.Println(err)
			http.Error(w, "comment validation failed", http.StatusBadRequest)
			return
		}

		// Verify user
		err = cmClient.verifyUser(ctx, c.verifyUserDisabled)
		if err != nil {
			log.Println(err)
			http.Error(w, "user not allowed to run command", http.StatusForbidden)
			return
		}

		// Extract args.
		err = cmClient.extractArgs(ctx)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not extract arguments", http.StatusBadRequest)
			return
		}

		// Post generated comment to GitHub pr.
		err = cmClient.generateAndPostComment(ctx)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not post comment to GitHub", http.StatusBadRequest)
			return
		}

		// Set label to Github pr.
		err = cmClient.postLabel(ctx)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not set label to GitHub", http.StatusBadRequest)
			return
		}

	default:
		log.Fatalln("only issue_comment event is supported")
	}
}

func (c commentMonitorConfig) postComment() error {
	// Setup commentMonitor client.
	cmClient := commentMonitorClient{
		allArgs:         make(map[string]string),
		commentTemplate: os.Getenv("COMMENT_TEMPLATE"),
	}
	// Setup GitHub client.
	owner := os.Getenv("GH_OWNER")
	repo := os.Getenv("GH_REPO")
	pr, err := strconv.Atoi(os.Getenv("GH_PR"))
	if err != nil {
		return fmt.Errorf("env var not set correctly")
	}
	ctx := context.Background()
	cmClient.ghClient, err = newGithubClient(ctx, owner, repo, pr)
	if err != nil {
		return fmt.Errorf("could not create GitHub client")
	}

	// Post generated comment to GitHub pr.
	err = cmClient.generateAndPostComment(ctx)
	if err != nil {
		return fmt.Errorf("could not post comment")
	}
	return nil
}
