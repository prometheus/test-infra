package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/google/go-github/v26/github"
	"golang.org/x/oauth2"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	app := kingpin.New(filepath.Base(os.Args[0]), "commentMonitor github comment extract\n ./commentMonitor -i /path/event.json -o /path \"^myregex$\"\nExample of comment template environment variable: COMMENT_TEMPLATE=\"The benchmark is starting. Your Github token is {{ index . \\\"GITHUB_TOKEN\\\" }}.\"")
	app.HelpFlag.Short('h')
	input := app.Flag("input", "path to event.json").Short('i').Default("/github/workflow/event.json").String()
	output := app.Flag("output", "path to write args to").Short('o').Default("/github/home").String()
	verifyUser := app.Flag("verify-user", "If set to true, a check, on whether the comment creator is a collaborator or a member of the group, will be executed.").Default("true").Bool()
	templateEnvVar := app.Flag("template-var", "Name of the environment variable that contains comment template.").Short('t').Default("COMMENT_TEMPLATE").String()
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

		owner := *e.GetRepo().Owner.Login
		repo := *e.GetRepo().Name
		prnumber := *e.GetIssue().Number

		// Github client for posting comments.
		clt := newClient(os.Getenv("GITHUB_TOKEN"))

		// Check author association.
		if *verifyUser {
			if err := memberValidation(*e.GetComment().AuthorAssociation); err != nil {
				log.Printf("author is not a member or collaborator")
				if err := postComment(clt, owner, repo, prnumber, fmt.Sprintf("Error: %s is not a group member nor a collaborator and cannot execute benchmarks.", *e.Sender.Login)); err != nil {
					log.Fatalln(err)
				}
				os.Exit(78)
			}
			log.Printf("author is member or collaborator")
		}

		// Validate comment.
		args, err := regexValidation(*regex, *e.GetComment().Body)
		if err != nil {
			log.Printf("matching command not found")
			os.Exit(78)
		}

		// Save args to file. Stores releaseVersion in ARG_0 and prnumber in ARG_1.
		args = append(args, strconv.Itoa(prnumber))
		writeArgs(args, *output)

		envVars := make(map[string]string)
		for _, e := range os.Environ() {
			tmp := strings.Split(e, "=")
			envVars[tmp[0]] = tmp[1]
		}

		var buf bytes.Buffer
		commentTemplate := template.Must(template.New("Comment").Parse(os.Getenv(*templateEnvVar)))
		if err := commentTemplate.Execute(&buf, envVars); err != nil {
			log.Fatalln(err)
		}

		if err := postComment(clt, owner, repo, prnumber, buf.String()); err != nil {
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
		// `args` are exported as environment variables so that they
		// can be used in the comment template.
		err = os.Setenv(filename, string(data))
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
