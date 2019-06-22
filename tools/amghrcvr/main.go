package main

import (
	"context"
	"encoding/json"
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

var (
	authfile       string
	defaultOwner   string
	defaultRepo    string
	ghClient       *github.Client
	issueSvc       *github.IssuesService
	ctx, cancelCtx = context.WithCancel(context.Background())
)

func processAlert(msg *notify.WebhookMessage) error {

	// add comment if firing
	if msg.Data.Status == "firing" {
		msgBody, err := formatIssueBody(msg)
		if err != nil {
			return err
		}

		prNo, err := getTargetPR(msg)
		if err != nil {
			return err
		}
		issueComment := github.IssueComment{Body: &msgBody}
		_, _, err = issueSvc.CreateComment(ctx, getTargetOwner(msg), getTargetRepo(msg), prNo, &issueComment)
		if err != nil {
			return err
		}
	}

	return nil
}

func handleWebhook(rw http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		log.Printf("Client used unsupported method: %s: %s", r.Method, r.RemoteAddr)
		rw.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	msg := &notify.WebhookMessage{}

	// decode the webhook request
	err := json.NewDecoder(r.Body).Decode(&msg)
	if err != nil {
		log.Println("Failed to decode json")
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	// Handle the webhook message.
	log.Printf("Handling alert: %s", id(msg))
	if err := processAlert(msg); err != nil {
		log.Printf("Failed to handle alert: %s: %s", id(msg), err)
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("Completed alert: %s", id(msg))
	rw.WriteHeader(http.StatusOK)

	return
}

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "amghrcvr - alertmanager github reciever")
	app.Flag("authfile", "path to github oauth token file").Default("/etc/github").StringVar(&authfile)
	app.Flag("org", "default org/owner").Required().StringVar(&defaultOwner)
	app.Flag("repo", "default repo").Required().StringVar(&defaultRepo)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	oauth2token, err := ioutil.ReadFile(authfile)
	if err != nil {
		log.Fatalln(err)
	}

	// setup ghClient
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(oauth2token)},
	)
	tc := oauth2.NewClient(ctx, ts)
	ghClient = github.NewClient(tc)
	issueSvc = ghClient.Issues
	log.Println("GitHub client successfully setup.")

	// start webhook server
	log.Printf("Started amghrcvr %v/%v as defaults.", defaultOwner, defaultRepo)
	http.HandleFunc("/hook", handleWebhook)
	log.Fatal(http.ListenAndServe(":8080", nil))

	<-ctx.Done()
}
