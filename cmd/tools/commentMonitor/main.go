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

var regex string
var input string
var output string

func writeArgs(arglist []string) {
	for i, arg := range arglist[1:] {
		data := []byte(arg)
		filename := fmt.Sprintf("ARG_%d", i)
		err := ioutil.WriteFile(filepath.Join(output, filename), data, 0644)
		log.Printf(filepath.Join(output, filename))
		if err != nil {
			log.Fatalln(err)
		}
	}
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
	} else {
		return []string{}, fmt.Errorf("invalid command")
	}
}

func postComment(client *github.Client, owner string, repo string, prnumber int, comment string) error {
	issueComment := &github.IssueComment{Body: github.String(comment)}
	issueComment, _, err := client.Issues.CreateComment(context.Background(), owner, repo, prnumber, issueComment)
	return err
}

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "simpleargs github comment extract")
	app.Flag("input", "path to event.json").Default("/github/workflow/event.json").StringVar(&input)
	app.Flag("output", "path to write args to").Default("/github/home").StringVar(&output)
	app.Arg("regex", "Regex pattern to match").Required().StringVar(&regex)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	// Github client for posting comments.
	token := os.Getenv("GITHUB_TOKEN")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Reading event.json.
	os.MkdirAll(output, os.ModePerm)
	data, err := ioutil.ReadFile(input)
	if err != nil {
		log.Fatalln(err)
	}

	// Parsing event.json.
	event, err := github.ParseWebHook("issue_comment", data)
	if err != nil {
		log.Fatalln("could not parse = %v\n", err)
	}

	switch e := event.(type) {
	case *github.IssueCommentEvent:
		// Check author association.
		if err := memberValidation(*e.GetComment().AuthorAssociation); err != nil {
			log.Printf("Author is not a member or collaborator")
			os.Exit(78)
		}
		log.Printf("Author is member or collaborator")

		// Validate comment.
		arglist, err := regexValidation(regex, *e.GetComment().Body)
		if err != nil {
			log.Printf("Matching command not found")
			os.Exit(78)
		}

		// Get parameters.
		owner := *e.GetRepo().Owner.Login
		repo := *e.GetRepo().Name
		prnumber := *e.GetIssue().Number
		releaseVersion := arglist[1]

		arglist = append(arglist, strconv.Itoa(prnumber))
		// Save args to file. Stores releaseVersion in ARG_0 and prnumber in ARG_1.
		writeArgs(arglist)

		// Posting benchmark start comment.
		comment := fmt.Sprintf(`Welcome to Prometheus Benchmarking Tool.

The two prometheus versions that will be compared are _**pr-%d**_ and _**%s**_

The logs can be viewed at the links provided in the GitHub check blocks at the end of this conversation

After successfull deployment, the benchmarking metrics can be viewed at :
- [prometheus-meta](%s/prometheus-meta) - label **{namespace="prombench-%d"}**
- [grafana](%s/grafana) - template-variable **"pr-number" : %d**

The Prometheus servers being benchmarked can be viewed at :
- PR - [prombench.prometheus.io/%d/prometheus-pr](%s/%d/prometheus-pr)
- %s - [prombench.prometheus.io/%d/prometheus-release](%s/%d/prometheus-release)

To stop the benchmark process comment **/benchmark cancel** .`, prnumber, releaseVersion, prombenchURL, prnumber, prombenchURL, prnumber, prnumber, prombenchURL, prnumber, releaseVersion, prnumber, prombenchURL, prnumber)

		if err := postComment(client, owner, repo, prnumber, comment); err != nil {
			log.Printf("%v+", err)
		}

	default:
		log.Fatalln("simpleargs only supports issue_comment event")
	}
}
