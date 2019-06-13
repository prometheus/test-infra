package main

import (
	"fmt"
	"github.com/google/go-github/v26/github"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
)

var regex string
var eventfile string
var writepath string

func writeArgs(groups []string) {
	for i, group := range groups[1:] {
		data := []byte(group)
		filename := fmt.Sprintf("ARG%d", i)
		err := ioutil.WriteFile(filepath.Join(writepath, filename), data, 0644)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "simpleargs github comment extract")
	app.Flag("eventfile", "path to event.json").Default("/github/workflow/event.json").StringVar(&eventfile)
	app.Flag("writepath", "path to write args to").Default("/github/home/simpleargs").StringVar(&writepath)
	app.Arg("regex", "Regex pattern to match").Required().StringVar(&regex)

	kingpin.MustParse(app.Parse(os.Args[1:]))

	os.MkdirAll(writepath, os.ModePerm)
	data, err := ioutil.ReadFile(eventfile)
	if err != nil {
		log.Fatalln(err)
	}

	event, err := github.ParseWebHook("issue_comment", data)
	if err != nil {
		log.Fatalln("could not parse = %s\n", err)
	}

	switch e := event.(type) {
	case *github.IssueCommentEvent:
		argRe := regexp.MustCompile(regex)
		if argRe.MatchString(*e.GetComment().Body) {
			groups := argRe.FindStringSubmatch(*e.GetComment().Body)
			writeArgs(groups)
		} else {
			log.Printf("matching command not found")
			os.Exit(78)
		}
	default:
		log.Fatalln("simpleargs only supports issue_comment event")
	}
}