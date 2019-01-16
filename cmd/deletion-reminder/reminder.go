package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/prombench/pkg/provider/k8s"
	"gopkg.in/alecthomas/kingpin.v2"
	apiCoreV1 "k8s.io/api/core/v1"
)

type client struct {
	githubClient *ghClient
	oauthFile    string
}

const (
	reminderCommentTemplate = `
	Benchmarking has been going on this PR for %v. 
	This is a reminder message to delete the benchmarking once you are done.
	To stop benchmarking, comment **/benchmark cancel** .
	`

	deletionCommentTemplate = `
	Benchmarking has been going on this PR for %v. 
	Stopping benchmarking now.
	`
	triggerCancelComment = "/benchmark cancel"

	remindAfterHours     = time.Duration(48) * time.Hour  // 2 days
	autoCancelAfterHours = time.Duration(240) * time.Hour // 10 days
)

func (c *client) checkNamespaceTimeout(namespace *apiCoreV1.Namespace, nsRunningHours map[string]time.Duration) error {
	name := namespace.Name

	if strings.HasPrefix(name, "prombench-") {
		prNumber, err := strconv.Atoi(name[len("prombench-"):len(name)])
		if err != nil {
			return err
		}

		runningDuration := time.Since(namespace.CreationTimestamp.Time)
		_, ok := nsRunningHours[name]
		if !ok {
			nsRunningHours[name] = remindAfterHours
		}

		// Send a github reminder after every `remindAfterHours`.
		// After `autoCancelAfterHours` has passed, prombot will automatically cancel the benchmark.
		if runningDuration >= nsRunningHours[name] {
			reminderComment := fmt.Sprintf(reminderCommentTemplate, runningDuration)
			log.Printf("Posting Deletion Reminder Comment for %d running for %v (>= %v)", prNumber, runningDuration, nsRunningHours[name])
			if err := c.githubClient.CreateComment(prNumber, reminderComment); err != nil {
				log.Printf("Error in posting deletion-reminder comment for %d : %v", prNumber, err)
			}

			if runningDuration >= autoCancelAfterHours {
				deletionComment := fmt.Sprintf(deletionCommentTemplate, runningDuration)
				log.Printf("Posting '/benchmark delete' comment for %d running for %v (>= %v)", prNumber, runningDuration, autoCancelAfterHours)
				if err := c.githubClient.CreateComment(prNumber, deletionComment); err != nil {
					log.Printf("Error in posting initiating-benchmark-deletion comment for %d : %v", prNumber, err)
				}

				if err := c.githubClient.CreateComment(prNumber, triggerCancelComment); err != nil {
					log.Printf("Error in posting triggerring-benchmark-deletion comment for %d : %v", prNumber, err)
				}
			}
			// Add duration for the next iteration.
			// If `autoCancelAfterHours` timeout has passed, then this is added to prevent repeated '/benchmark cancel' comments
			// while namespace is being deleted
			nsRunningHours[name] = time.Duration(int64(nsRunningHours[name].Hours()+remindAfterHours.Hours())) * time.Hour
		} else {
			log.Printf("%d has been running for %v (<= %v)", prNumber, runningDuration, nsRunningHours[name])
		}
	}
	return nil
}

func (c *client) run(*kingpin.ParseContext) error {
	log.Printf("Starting Prombench-Reminder-Tool")

	k, k8serr := k8s.New(context.Background(), nil)
	if k8serr != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(k8serr, "Error creating k8s client inside the k8s cluster."))
		os.Exit(2)
	}

	gh, ghErr := NewGHClient(c.oauthFile)
	if ghErr != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(ghErr, "Error creating GitHub client."))
		os.Exit(2)
	}

	c.githubClient = gh
	nsRunningHours := make(map[string]time.Duration)

	for {
		runningNs, err := k.GetNameSpaces()
		if err != nil {
			fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error fetching namespaces list."))
		} else {
			runningNsMap := make(map[string]struct{}) //used to remove deleted namespaces from nsRunningHours

			for _, namespace := range runningNs {
				if namespace.Status.Phase == apiCoreV1.NamespaceActive {
					runningNsMap[namespace.Name] = struct{}{}
					if err := c.checkNamespaceTimeout(&namespace, nsRunningHours); err != nil {
						fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error checking timeout for namespace %s.", namespace.Name))
					}
				}
			}

			// removing deleted namespaces from nsRunningHours
			for name := range nsRunningHours {
				if _, ok := runningNsMap[name]; !ok {
					delete(nsRunningHours, name)
				}
			}
		}
		time.Sleep(5 * time.Minute)
	}
}

func main() {

	app := kingpin.New(filepath.Base(os.Args[0]), "The Prombench-Reminder tool")
	app.HelpFlag.Short('h')

	var s client

	k8sApp := app.Command("remind", "Remind a maintainer that a deployment is still running. \nex: ./deletion-reminder remind -f /etc/github/oauth").
		Action(s.run)
	k8sApp.Flag("file", "File containing GitHub oauth token.").
		Required().
		Short('f').
		ExistingFileVar(&s.oauthFile)

	if _, err := app.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}
}
