package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
)

const prombenchURL = "http://prombench.prometheus.io"

func newClient(token string) *github.Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	clt := github.NewClient(tc)
	return clt
}
func memberValidation(authorAssociation string) error {
	if (authorAssociation != "COLLABORATOR") && (authorAssociation != "MEMBER") {
		return fmt.Errorf("not a member or collaborator")
	}
	return nil
}

func regexValidation(regex string, comment string) ([]string, error) {
	argRe := regexp.MustCompile(regex)
	if argRe.MatchString(comment) {
		arglist := argRe.FindStringSubmatch(comment)
		return arglist, nil
	}
	return []string{}, fmt.Errorf("invalid command")
}

func writeArgs(args []string, output string) {
	for i, arg := range args[1:] {
		data := []byte(arg)
		filename := fmt.Sprintf("ARG_%v", i)
		err := ioutil.WriteFile(filepath.Join(output, filename), data, 0644)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf(filepath.Join(output, filename))
	}
}

func postComment(client *github.Client, owner string, repo string, prnumber int, comment string) error {
	issueComment := &github.IssueComment{Body: github.String(comment)}
	_, _, err := client.Issues.CreateComment(context.Background(), owner, repo, prnumber, issueComment)
	return err
}

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "commentMonitor github comment extract")
	app.HelpFlag.Short('h')
	input := app.Flag("input", "path to event.json").Short('i').Default("/github/workflow/event.json").String()
	output := app.Flag("output", "path to write args to").Short('o').Default("/github/home").String()
	regex := app.Arg("regex", "Regex pattern to match").Required().String()

	kingpin.MustParse(app.Parse(os.Args[1:]))

	// Reading event.json.
	err := os.MkdirAll(*output, os.ModePerm)
	if err != nil {
		log.Fatalln(err)
	}
	data, err := ioutil.ReadFile(*input)
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
		// Check author association.
		if err := memberValidation(*e.GetComment().AuthorAssociation); err != nil {
			log.Printf("author is not a member or collaborator")
			os.Exit(78)
		}
		log.Printf("author is member or collaborator")

		// Validate comment.
		args, err := regexValidation(*regex, *e.GetComment().Body)
		if err != nil {
			log.Printf("matching command not found")
			os.Exit(78)
		}

		// Get parameters.
		releaseVersion := args[1]
		owner := *e.GetRepo().Owner.Login
		repo := *e.GetRepo().Name
		prnumber := *e.GetIssue().Number

		// Save args to file. Stores releaseVersion in ARG_0 and prnumber in ARG_1.
		args = append(args, strconv.Itoa(prnumber))
		writeArgs(args, *output)

		// Posting benchmark start comment.
		comment := fmt.Sprintf(`Welcome to Prometheus Benchmarking Tool.

The two prometheus versions that will be compared are _**pr-%v**_ and _**%v**_

The logs can be viewed at the links provided in the GitHub check blocks at the end of this conversation

After successfull deployment, the benchmarking metrics can be viewed at :
- [prometheus-meta](%v/prometheus-meta) - label **{namespace="prombench-%v"}**
- [grafana](%v/grafana) - template-variable **"pr-number" : %v**

The Prometheus servers being benchmarked can be viewed at :
- PR - [prombench.prometheus.io/%v/prometheus-pr](%v/%v/prometheus-pr)
- %v - [prombench.prometheus.io/%v/prometheus-release](%v/%v/prometheus-release)

To stop the benchmark process comment **/benchmark cancel** .`, prnumber, releaseVersion, prombenchURL, prnumber, prombenchURL, prnumber, prnumber, prombenchURL, prnumber, releaseVersion, prnumber, prombenchURL, prnumber)

		// Github client for posting comments.
		clt := newClient(os.Getenv("GITHUB_TOKEN"))

		if err := postComment(clt, owner, repo, prnumber, comment); err != nil {
			log.Fatalln(err)
		}

		// Setting benchmark label.
		benchmarkLabel := []string{"benchmark"}
		if _, _, err := clt.Issues.AddLabelsToIssue(context.Background(), owner, repo, prnumber, benchmarkLabel); err != nil {
			log.Fatalln(err)
		}

	default:
		log.Fatalln("simpleargs only supports issue_comment event")
	}
}
