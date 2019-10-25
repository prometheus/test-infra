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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v3"
)

type commentMonitorConfig struct {
	inputFilePath      string
	outputDirPath      string
	eventMapFilePath   string
	regexString        string
	verifyUserDisabled bool
	webhook            bool
	eventMap           webhookEventMaps
}

type webhookEventMap struct {
	EventType       string `yaml:"event_type"`
	CommentTemplate string `yaml:"comment_template"`
	RegexString     string `yaml:"regex_string"`
}

type webhookEventMaps []webhookEventMap

func main() {
	log.SetFlags(log.Ltime | log.Lshortfile)
	cmConfig := commentMonitorConfig{}

	app := kingpin.New(filepath.Base(os.Args[0]), `commentMonitor github comment extract
	./commentMonitor -i /path/event.json -o /path "^myregex$"
	Example of comment template environment variable:
	COMMENT_TEMPLATE="The benchmark is starting. Your Github token is {{ index . "SOME_VAR" }}."`)
	app.HelpFlag.Short('h')
	app.Flag("input", "path to event.json").
		Short('i').
		Default("/github/workflow/event.json").
		StringVar(&cmConfig.inputFilePath)
	app.Flag("output", "path to write args to").
		Short('o').
		Default("/github/home/commentMonitor").
		StringVar(&cmConfig.outputDirPath)
	app.Flag("no-verify-user", "disable verifying user").
		BoolVar(&cmConfig.verifyUserDisabled)
	app.Flag("webhook", "enable webhook mode").
		BoolVar(&cmConfig.webhook)
	app.Flag("eventmap", "Filepath to eventmap file.").
		Default("./eventmap.yaml").
		StringVar(&cmConfig.eventMapFilePath)
	app.Arg("regex", "Regex pattern to match").
		StringVar(&cmConfig.regexString)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	if cmConfig.webhook {
		// get eventmap file.
		data, err := ioutil.ReadFile(cmConfig.eventMapFilePath)
		if err != nil {
			log.Fatalln(err)
		}
		err = yaml.Unmarshal(data, &cmConfig.eventMap)
		if err != nil {
			log.Fatalf("cannot unmarshal data: %v", err)
		}
		// run webhook server.
		http.HandleFunc("/", cmConfig.webhookExtract)
		log.Fatal(http.ListenAndServe(":8080", nil))
	}
	cmConfig.eventJSONExtract()
}

func newGithubClient(ctx context.Context, e *github.IssueCommentEvent) githubClient {
	ghToken := os.Getenv("GITHUB_TOKEN")
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: ghToken})
	tc := oauth2.NewClient(ctx, ts)
	return githubClient{
		clt:               github.NewClient(tc),
		owner:             *e.GetRepo().Owner.Login,
		repo:              *e.GetRepo().Name,
		pr:                *e.GetIssue().Number,
		author:            *e.Sender.Login,
		authorAssociation: *e.GetComment().AuthorAssociation,
		commentBody:       *e.GetComment().Body,
	}
}

func (c commentMonitorConfig) webhookExtract(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "unable to read webhook body", http.StatusBadRequest)
	}

	// Setup commentMonitor client.
	cmClient := commentMonitorClient{
		allArgs: make(map[string]string),
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
		cmClient.ghClient = newGithubClient(ctx, e)

		// Validate regex.
		err := cmClient.validateRegex(c.regexString)
		if err != nil {
			log.Println(err)
			http.Error(w, "comment validation failed", http.StatusBadRequest)
		}

		// Verify user
		err = cmClient.verifyUser(ctx, c.verifyUserDisabled)
		if err != nil {
			log.Println(err)
			http.Error(w, "comment validation failed", http.StatusForbidden)
		}

		// Extract args.
		err = cmClient.extractArgs(ctx, c.outputDirPath)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not extract arguments", http.StatusBadRequest)
		}

		// Post generated comment to GitHub pr.
		err = cmClient.generateAndPostComment(ctx)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not post comment to GitHub", http.StatusBadRequest)
		}

		// Set label to Github pr.
		err = cmClient.postLabel(ctx)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not set label to GitHub", http.StatusBadRequest)
		}

	default:
		log.Fatalln("only issue_comment event is supported")
	}
}

func (c commentMonitorConfig) eventJSONExtract() {
	err := os.MkdirAll(c.outputDirPath, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	data, err := ioutil.ReadFile(c.inputFilePath)
	if err != nil {
		log.Fatalln(err)
	}

	// Setup commentMonitor client.
	cmClient := commentMonitorClient{
		allArgs:         make(map[string]string),
		commentTemplate: os.Getenv("COMMENT_TEMPLATE"),
	}

	// Parse event.json.
	event, err := github.ParseWebHook("issue_comment", data)
	if err != nil {
		log.Fatalln(err)
	}

	switch e := event.(type) {
	case *github.IssueCommentEvent:

		// Setup github client.
		ctx := context.Background()
		cmClient.ghClient = newGithubClient(ctx, e)

		// Validate regex.
		err := cmClient.validateRegex(c.regexString)
		if err != nil {
			log.Fatalln(err)
		}

		// Verify user.
		err = cmClient.verifyUser(ctx, c.verifyUserDisabled)
		if err != nil {
			log.Fatalln(err)
		}

		// Extract args.
		err = cmClient.extractArgs(ctx, c.outputDirPath)
		if err != nil {
			log.Fatalln(err)
		}

		// Post generated comment to GitHub pr.
		err = cmClient.generateAndPostComment(ctx)
		if err != nil {
			log.Fatalln(err)
		}

		// Set label to Github pr.
		err = cmClient.postLabel(ctx)
		if err != nil {
			log.Fatalln(err)
		}

	default:
		log.Fatalln("only issue_comment event is supported")
	}
}
