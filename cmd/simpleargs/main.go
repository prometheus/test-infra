package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/google/go-github/github"
	"gopkg.in/alecthomas/kingpin.v2"
)

var regex string
var eventfile string
var writepath string

type roundTripper struct {
	accessToken string
}

func (rt roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", fmt.Sprintf("token %s", rt.accessToken))
	return http.DefaultTransport.RoundTrip(r)
}
func writeArgs(groups []string) {
	for i, group := range groups[1:] {
		data := []byte(group)
		filename := fmt.Sprintf("ARG_%d", i)
		log.Printf("In write args")
		err := ioutil.WriteFile(filepath.Join(writepath, filename), data, 0644)
		log.Printf(filepath.Join(writepath, filename))
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func writeArg(groups string, filetype string) {
	data := []byte(groups)
	filename := fmt.Sprintf("ARG_%s", filetype)
	log.Printf("In write arg")
	err := ioutil.WriteFile(filepath.Join(writepath, filename), data, 0644)
	log.Printf(filepath.Join(writepath, filename))
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "simpleargs github comment extract")
	app.Flag("eventfile", "path to event.json").Default("/github/workflow/event.json").StringVar(&eventfile)
	app.Flag("writepath", "path to write args to").Default("/github/home").StringVar(&writepath)
	app.Arg("regex", "Regex pattern to match").Required().StringVar(&regex)
	kingpin.MustParse(app.Parse(os.Args[1:]))
	log.Printf("After flag declaration")
	var err error

	//Github client for posting comments
	token := os.Getenv("GITHUB_TOKEN")
	http.DefaultClient.Transport = roundTripper{token}
	client := github.NewClient(http.DefaultClient)

	//Decoding and saving service account authentication file
	paths := os.Getenv("HOME")
	paths = paths + "/auth.json"
	output, err := os.Create(paths)
	if err != nil {
		panic(err)
	}
	defer output.Close()
	strings := os.Getenv("AUTH_FILE")
	decoder, err := base64.StdEncoding.DecodeString(strings)
	if err != nil {
		panic(err)
	}
	if _, err := output.Write(decoder); err != nil {
		panic(err)
	}
	fmt.Println("storing base64 decoded auth file")

	//Reading event.json
	os.MkdirAll(writepath, os.ModePerm)
	data, err := ioutil.ReadFile(eventfile)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("after reading event.json")

	//Parsing event.json
	event, err := github.ParseWebHook("issue_comment", data)
	if err != nil {
		log.Fatalln("could not parse = %v\n", err)
	}
	log.Printf("Parsing event.json")

	//Checking author association and saving args to file
	switch e := event.(type) {
	case *github.IssueCommentEvent:
		if (*e.GetComment().AuthorAssociation != "COLLABORATOR") && (*e.GetComment().AuthorAssociation != "MEMBER") {
			log.Printf("author is not a member or collaborator")
			os.Exit(78)
		} else {
			log.Printf("author is member or collaborator")

			owner := *e.GetRepo().Owner.Login
			repo := *e.GetRepo().Name
			number := *e.GetIssue().Number
			log.Printf(owner)
			log.Printf(repo)
			log.Printf("%d", number)
			writeArg(strconv.Itoa(number), "pr") //writing PR number to file

			argRe := regexp.MustCompile(regex)
			if argRe.MatchString(*e.GetComment().Body) {
				groups := argRe.FindStringSubmatch(*e.GetComment().Body)
				log.Printf("%v+", groups)
				writeArgs(groups) //writing version to file
				log.Printf("regex comment")

				//Posting benchmark start comment
				issueComment := &github.IssueComment{Body: github.String("Benchmarking is starting")}
				issueComment, _, err = client.Issues.CreateComment(context.Background(), owner, repo, number, issueComment)
				if err != nil {
					fmt.Printf("%v+", err)
				}

			} else {
				log.Printf("matching command not found")
				os.Exit(78)
			}
		}
	default:
		log.Fatalln("simpleargs only supports issue_comment event")
	}
}
