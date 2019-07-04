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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/go-github/v26/github"
	"github.com/prometheus/alertmanager/notify"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
)

type ghWebhookReceiverConfig struct {
	authfile     string
	defaultOwner string
	defaultRepo  string
	portNo       string
}

type ghWebhookReceiver struct {
	ghClient *github.Client
	cfg      ghWebhookReceiverConfig
}

type ghWebhookHandler struct {
	client *ghWebhookReceiver
}

func main() {
	cfg := ghWebhookReceiverConfig{}

	app := kingpin.New(filepath.Base(os.Args[0]), "alertmanager github webhook receiver")
	app.Flag("authfile", "path to github oauth token file").Default("/etc/github/oauth").StringVar(&cfg.authfile)
	app.Flag("org", "default org/owner").Required().StringVar(&cfg.defaultOwner)
	app.Flag("repo", "default repo").Required().StringVar(&cfg.defaultRepo)
	app.Flag("port", "port number to run the server in").Default("8080").StringVar(&cfg.portNo)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	client, err := newGhWebhookReceiver(cfg)
	if err != nil {
		log.Fatalf("failed to create GitHub Webhook Receiver client: %v", err)
	}

	serveWebhook(client)
}

func (hl ghWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("unsupported request method: %v: %v", r.Method, r.RemoteAddr)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	msg := &notify.WebhookMessage{}
	ctx := r.Context()

	// Decode the webhook request.
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		log.Println("failed to decode webhook data")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle the webhook message.
	log.Printf("handling alert: %v", alertID(msg))
	if err := hl.client.processAlert(ctx, msg); err != nil {
		log.Printf("failed to handle alert: %v: %v", alertID(msg), err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("completed alert: %v", alertID(msg))
	w.WriteHeader(http.StatusOK)
}

func newGhWebhookReceiver(cfg ghWebhookReceiverConfig) (*ghWebhookReceiver, error) {
	oauth2token, err := ioutil.ReadFile(cfg.authfile)
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

// processAlert formats and posts the comment to github and returns nil if successful.
func (g ghWebhookReceiver) processAlert(ctx context.Context, msg *notify.WebhookMessage) error {

	msgBody, err := formatIssueBody(msg)
	if err != nil {
		return err
	}
	issueComment := github.IssueComment{Body: &msgBody}

	prNum, err := getTargetPR(msg)
	if err != nil {
		return err
	}

	_, _, err = g.ghClient.Issues.CreateComment(ctx,
		g.getTargetOwner(msg), g.getTargetRepo(msg), prNum, &issueComment)
	if err != nil {
		return err
	}

	return nil
}

func serveWebhook(client *ghWebhookReceiver) {
	hl := ghWebhookHandler{client}
	http.Handle("/hook", hl)
	log.Printf("finished setting up gh client. starting amgh with %v/%v as defaults",
		client.cfg.defaultOwner, client.cfg.defaultRepo)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", client.cfg.portNo), nil))
}
