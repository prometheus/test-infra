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
	eventMapFilePath   string
	whSecretFilePath   string
	commandPrefixes    string
	whSecret           []byte
	eventMap           webhookEventMaps
	port               string
}

// Structure of eventmap.yml file.
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
	app.Flag("command-prefixes", `Comma separated list of command prefixes. Eg."/prombench,/funcbench" `).
		Required().
		StringVar(&cmConfig.commandPrefixes)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	mux := http.NewServeMux()
	mux.HandleFunc("/", cmConfig.webhookExtract)
	log.Println("Server is ready to handle requests at", cmConfig.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", cmConfig.port), mux))
}

func (c *commentMonitorConfig) loadConfig() error {
	// Get eventmap file.
	data, err := ioutil.ReadFile(c.eventMapFilePath)
	if err != nil {
		return err
	}
	err = yaml.UnmarshalStrict(data, &c.eventMap)
	if err != nil {
		return fmt.Errorf("cannot unmarshal data: %v", err)
	}
	if len(c.eventMap) == 0 {
		return fmt.Errorf("eventmap empty")
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

func checkCommandPrefix(command, prefixStrings string) bool {
	prefixes := strings.Split(prefixStrings, ",")
	for _, p := range prefixes {
		i := strings.Index(command, p)
		if i == 0 {
			return true
		}
	}
	return false
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
		cmClient.ghClient, err = newGithubClient(ctx, e)
		if err != nil {
			log.Println(err)
			http.Error(w, "could not create GitHub client", http.StatusBadRequest)
			return
		}

		// Strip whitespace.
		command := extractCommand(cmClient.ghClient.commentBody)

		// test-infra command check.
		if !checkCommandPrefix(command, c.commandPrefixes) {
			http.Error(w, "comment validation failed", http.StatusOK)
			return
		}

		// Validate regex.
		if !cmClient.validateRegex(command) {
			log.Println("invalid command syntax: ", command)
			if err := cmClient.ghClient.postComment(ctx, "command syntax invalid"); err != nil {
				log.Printf("%v : couldn't post comment", err)
			}
			http.Error(w, "command syntax invalid", http.StatusBadRequest)
			return
		}

		// Verify user.
		err = cmClient.verifyUser(ctx, c.verifyUserDisabled)
		if err != nil {
			log.Println(err)
			http.Error(w, "user not allowed to run command", http.StatusForbidden)
			return
		}

		// Extract args.
		err = cmClient.extractArgs(ctx, command)
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
		log.Println("only issue_comment event is supported")
	}
}
