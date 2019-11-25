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

	"github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v3"
)

type commentMonitorConfig struct {
	verifyUserDisabled bool
	eventMapFilePath   string
	whSecretFilePath   string
	whSecret           []byte
	eventMap           webhookEventMaps
	port               string
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
	app.Flag("webhooksecretfile", "path to webhook secret file").
		Default("./whsecret").
		StringVar(&cmConfig.whSecretFilePath)
	app.Flag("no-verify-user", "disable verifying user").
		BoolVar(&cmConfig.verifyUserDisabled)
	app.Flag("eventmap", "Filepath to eventmap file.").
		Default("./eventmap.yml").
		StringVar(&cmConfig.eventMapFilePath)
	app.Flag("port", "port number to run webhook in.").
		Default("8080").
		StringVar(&cmConfig.port)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	err := cmConfig.loadConfig()
	if err != nil {
		log.Fatalln(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", cmConfig.webhookExtract)
	mux.HandleFunc("/-/reload", cmConfig.reloadConfig)
	log.Println("Server is ready to handle requests at", cmConfig.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", cmConfig.port), mux))
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

func (c *commentMonitorConfig) loadConfig() error {
	// Get eventmap file.
	data, err := ioutil.ReadFile(c.eventMapFilePath)
	if err != nil {
		return err
	}
	// TODO: Do strict checking.
	err = yaml.Unmarshal(data, &c.eventMap)
	if err != nil {
		return fmt.Errorf("cannot unmarshal data: %v", err)
	}
	if len(c.eventMap) == 0 {
		return fmt.Errorf("eventmap empty")
	}
	// Get webhook secret.
	c.whSecret, err = ioutil.ReadFile(c.whSecretFilePath)
	fmt.Println(c.whSecret)
	if err != nil {
		return err
	}
	return nil
}

func (c *commentMonitorConfig) reloadConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "unsupported method", http.StatusBadRequest)
		return
	}
	err := c.loadConfig()
	if err != nil {
		http.Error(w, "reload unsuccessful", http.StatusInternalServerError)
	}
	w.WriteHeader(http.StatusCreated)
}

func (c *commentMonitorConfig) webhookExtract(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	payload, err := github.ValidatePayload(r, c.whSecret)
	if err != nil {
		log.Println(err)
		http.Error(w, "unable to read webhook body", http.StatusBadRequest)
		return
	}

	// Setup commentMonitor client.
	cmClient := commentMonitorClient{
		allArgs:  make(map[string]string),
		eventMap: c.eventMap,
	}

	// Parse webhook event.
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		log.Println(err)
		http.Error(w, "unable to parse webhook", http.StatusBadRequest)
		return
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
		if !cmClient.validateRegex() {
			//log.Println(err) // Don't log on failure.
			http.Error(w, "comment validation failed", http.StatusOK)
			return
		}

		// Verify user.
		err = cmClient.verifyUser(ctx, c.verifyUserDisabled)
		if err != nil {
			log.Println(err)
			http.Error(w, "user not allowed to run command", http.StatusForbidden)
			return
		}

		// Get the last commit sha from PR.
		cmClient.allArgs["LAST_COMMIT_SHA"], err = cmClient.ghClient.getLastCommitSHA(ctx)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not fetch sha", http.StatusBadRequest)
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

		// Set label to GitHub pr.
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
