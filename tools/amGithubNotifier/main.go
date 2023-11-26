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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-github/v29/github"
	"github.com/prometheus/alertmanager/notify/webhook"
	"github.com/prometheus/alertmanager/template"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
)

type ghWebhookReceiverConfig struct {
	authFile string
	org      string
	repo     string
	portNo   string
	dryRun   bool
}

type ghWebhookReceiver struct {
	ghClient *github.Client
	cfg      ghWebhookReceiverConfig
}

type ghWebhookHandler struct {
	client *ghWebhookReceiver
}

func main() {
	/*
		Example `alerts.rules.yml`:
		```
		groups:
		- name: groupname
		  rules:
		  - alert: alertname
		    expr: up == 0
		    labels:
		      severity: info
		      prNum: '{{ $labels.prNum }}'
		      org: prometheus
		      repo: prombench
		    annotations:
			  description: 'description of the alert'
		```
	*/
	log.SetFlags(log.Ltime | log.Lshortfile)
	cfg := ghWebhookReceiverConfig{}

	app := kingpin.New(filepath.Base(os.Args[0]), `alertmanager github webhook receiver
	Example: ./amGithubNotifier --org=prometheus --repo=prometheus --port=8080

	Note: All alerts sent to amGithubNotifier must have the prNum label and description
	annotation, org and repo labels are optional but will take precedence over cli args
	if provided.
	`)
	app.Flag("authfile", "path to github oauth token file").Default("/etc/github/oauth").StringVar(&cfg.authFile)
	app.Flag("org", "name of the org").Required().StringVar(&cfg.org)
	app.Flag("repo", "name of the repo").Required().StringVar(&cfg.repo)
	app.Flag("port", "port number to run the server in").Default("8080").StringVar(&cfg.portNo)
	app.Flag("dryrun", "dry run for github api").BoolVar(&cfg.dryRun)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	client, err := newGhWebhookReceiver(cfg)
	if err != nil {
		log.Fatalf("failed to create GitHub Webhook Receiver client: %v", err)
	}

	hl := ghWebhookHandler{client}
	http.Handle("/hook", hl)
	log.Printf("finished setting up gh client. starting amGithubNotifier with %v/%v",
		client.cfg.org, client.cfg.repo)
	err = http.ListenAndServe(fmt.Sprintf(":%v", client.cfg.portNo), nil)
	if err != nil {
		log.Fatal("amGithubNotifier exited unexpectedly")
	}
}

func (hl ghWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("unsupported request method: %v: %v", r.Method, r.RemoteAddr)
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	msg := &webhook.Message{}
	ctx := r.Context()

	// Decode the webhook request.
	err := json.NewDecoder(r.Body).Decode(msg)
	if err != nil {
		log.Println("failed to decode webhook data")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Handle the webhook message.
	log.Printf("handling alert: %v", alertID(msg))
	if _, err := hl.client.processAlerts(ctx, msg); err != nil {
		log.Printf("failed to handle alert: %v: %v", alertID(msg), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("completed alert: %v", alertID(msg))
	w.WriteHeader(http.StatusOK)
}

func newGhWebhookReceiver(cfg ghWebhookReceiverConfig) (*ghWebhookReceiver, error) {
	if cfg.dryRun {
		return &ghWebhookReceiver{
			ghClient: github.NewClient(nil),
			cfg:      cfg,
		}, nil
	}

	// Add github token.
	oauth2token, err := os.ReadFile(cfg.authFile)
	if err != nil {
		return nil, err
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(oauth2token)},
	)
	ctx := context.Background()
	tc := oauth2.NewClient(ctx, ts)

	return &ghWebhookReceiver{
		ghClient: github.NewClient(tc),
		cfg:      cfg,
	}, nil
}

// processAlert formats and posts the alert to GitHub.
func (g ghWebhookReceiver) processAlert(ctx context.Context, alert template.Alert) (string, error) {
	msgBody, err := formatIssueCommentBody(alert)
	if err != nil {
		return "", err
	}
	issueComment := github.IssueComment{Body: &msgBody}

	prNum, err := getTargetPR(alert)
	if err != nil {
		return "", err
	}

	if g.cfg.dryRun {
		return msgBody, err
	}
	_, _, err = g.ghClient.Issues.CreateComment(ctx,
		g.getTargetOrg(alert), g.getTargetRepo(alert), prNum, &issueComment)

	return msgBody, err
}

func (g ghWebhookReceiver) processAlerts(ctx context.Context, msg *webhook.Message) ([]string, error) {
	var alertcomments []string

	// Each alert will have its own comment.
	for _, a := range msg.Alerts {
		alertcomment, err := g.processAlert(ctx, a)
		if err != nil {
			return nil, err
		}
		alertcomments = append(alertcomments, alertcomment)
	}
	return alertcomments, nil
}
