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
	"strings"

	"github.com/google/go-github/v29/github"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

type commentMonitorConfig struct {
	verifyUserDisabled bool
	configFilePath     string
	whSecretFilePath   string
	whSecret           []byte
	configFile         configFile
	port               string
}

type commandPrefix struct {
	Prefix       string `yaml:"prefix"`
	HelpTemplate string `yaml:"help_template"`
}

type webhookEvent struct {
	EventType       string `yaml:"event_type"`
	CommentTemplate string `yaml:"comment_template"`
	RegexString     string `yaml:"regex_string"`
	Label           string `yaml:"label"`
}

type configFile struct {
	Prefixes      []commandPrefix `yaml:"prefixes"`
	WebhookEvents []webhookEvent  `yaml:"events"`
}

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
	app.Flag("config", "Filepath to config file.").
		Default("./config.yml").
		StringVar(&cmConfig.configFilePath)
	app.Flag("port", "port number to run webhook in.").
		Default("8080").
		StringVar(&cmConfig.port)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	mux := http.NewServeMux()
	mux.HandleFunc("/", cmConfig.webhookExtract)
	log.Println("Server is ready to handle requests at", cmConfig.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", cmConfig.port), mux))
}

func (c *commentMonitorConfig) loadConfig() error {
	// Get config file.
	data, err := ioutil.ReadFile(c.configFilePath)
	if err != nil {
		return err
	}
	err = yaml.UnmarshalStrict(data, &c.configFile)
	if err != nil {
		return fmt.Errorf("cannot unmarshal data: %v", err)
	}
	if len(c.configFile.WebhookEvents) == 0 || len(c.configFile.Prefixes) == 0 {
		return fmt.Errorf("empty eventmap or prefix list")
	}
	// Get webhook secret.
	c.whSecret, err = ioutil.ReadFile(c.whSecretFilePath)
	if err != nil {
		return err
	}
	return nil
}

func extractCommand(s string) string {
	s = strings.TrimLeft(s, "\r\n\t ")
	if i := strings.Index(s, "\n"); i != -1 {
		s = s[:i]
	}
	s = strings.TrimRight(s, "\r\n\t ")
	return s
}

func (c *commentMonitorConfig) webhookExtract(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// Load config on every request.
	err := c.loadConfig()
	if err != nil {
		log.Println(err)
		http.Error(w, "comment-monitor configuration incorrect", http.StatusInternalServerError)
		return
	}

	// Validate payload.
	payload, err := github.ValidatePayload(r, c.whSecret)
	if err != nil {
		log.Println(err)
		http.Error(w, "unable to read webhook body", http.StatusBadRequest)
		return
	}

	// Setup commentMonitor client.
	cmClient := commentMonitorClient{
		allArgs:  make(map[string]string),
		events:   c.configFile.WebhookEvents,
		prefixes: c.configFile.Prefixes,
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

		if *e.Action != "created" {
			http.Error(w, "issue_comment type must be 'created'", http.StatusOK)
			return
		}

		// Setup github client.
		ctx := context.Background()
		cmClient.ghClient, err = newGithubClient(ctx, e)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not create GitHub client", http.StatusBadRequest)
			return
		}

		// Strip whitespace.
		command := extractCommand(cmClient.ghClient.commentBody)

		// Command check.
		if !cmClient.checkCommandPrefix(command) {
			http.Error(w, "comment validation failed", http.StatusOK)
			return
		}

		// Validate regex.
		if !cmClient.validateRegex(command) {
			log.Println("invalid command syntax: ", command)
			err = cmClient.generateAndPostErrorComment()
			if err != nil {
				log.Println(err)
				http.Error(w, "could not post comment to GitHub", http.StatusBadRequest)
				return
			}
			http.Error(w, "command syntax invalid", http.StatusBadRequest)
			return
		}

		// Verify user.
		err = cmClient.verifyUser(c.verifyUserDisabled)
		if err != nil {
			log.Println(err)
			http.Error(w, "user not allowed to run command", http.StatusForbidden)
			return
		}

		// Extract args.
		err = cmClient.extractArgs(command)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not extract arguments", http.StatusBadRequest)
			return
		}

		// Post generated comment to GitHub pr.
		err = cmClient.generateAndPostSuccessComment()
		if err != nil {
			log.Println(err)
			http.Error(w, "could not post comment to GitHub", http.StatusBadRequest)
			return
		}

		// Set label to GitHub pr.
		err = cmClient.postLabel()
		if err != nil {
			log.Println(err)
			http.Error(w, "could not set label to GitHub", http.StatusBadRequest)
			return
		}

	default:
		log.Println("only issue_comment event is supported")
	}
}
