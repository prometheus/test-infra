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
)

type commentMonitorClient struct {
	ghClient        githubClient
	allArgs         map[string]string
	regex           *regexp.Regexp
	eventMap        webhookEventMaps
	eventType       string
	commentTemplate string
}

// Validate comment if regexString provided.
func (c commentMonitorClient) validateRegex(regexString string) error {
	// For webhook.
	if len(c.eventMap) != 0 {
		for _, e := range c.eventMap {
			c.regex = regexp.MustCompile(e.RegexString)
			if c.regex.MatchString(c.ghClient.commentBody) {
				c.commentTemplate = e.CommentTemplate
				c.eventType = e.EventType
				log.Println("comment validation successful")
				return nil
			}
		}
		return fmt.Errorf("matching command not found. comment validation failed")
	}

	// For one-time run.
	if regexString != "" {
		c.regex = regexp.MustCompile(regexString)
		if !c.regex.MatchString(c.ghClient.commentBody) {
			return fmt.Errorf("matching command not found. comment validation failed")
		}
		log.Println("comment validation successful")
	}
	return nil
}

// Verify if user is allowed to perform activity.
func (c commentMonitorClient) verifyUser(ctx context.Context, verifyUserDisabled bool) error {
	if !verifyUserDisabled {
		var allowed bool
		allowedAssociations := []string{"COLLABORATOR", "MEMBER", "OWNER"}
		for _, a := range allowedAssociations {
			if a == c.ghClient.authorAssociation {
				allowed = true
			}
		}
		if !allowed {
			b := fmt.Sprintf("@%s is not a org member nor a collaborator and cannot execute benchmarks.", c.ghClient.author)
			if err := c.ghClient.postComment(ctx, b); err != nil {
				return fmt.Errorf("%v : couldn't post comment", err)
			}
			return fmt.Errorf("author is not a member or collaborator")
		}
		log.Println("author is a member or collaborator")
	}
	return nil
}

// Extract args if regexString provided.
func (c commentMonitorClient) extractArgs(ctx context.Context, outputDirPath string) error {
	if c.regex != nil {
		// Add comment arguments.
		commentArgs := c.regex.FindStringSubmatch(c.ghClient.commentBody)[1:]
		commentArgsNames := c.regex.SubexpNames()[1:]
		for i, argName := range commentArgsNames {
			if argName == "" {
				return fmt.Errorf("using named groups is mandatory")
			}
			c.allArgs[argName] = commentArgs[i]
		}

		// Add non-comment arguments if any.
		c.allArgs["PR_NUMBER"] = strconv.Itoa(c.ghClient.pr)

		if len(c.eventMap) != 0 {
			err := c.ghClient.createRepositoryDispatch(ctx, c.eventType, c.allArgs)
			if err != nil {
				return fmt.Errorf("%v: could not create repository_dispatch event", err)
			}
		} else {
			err := c.writeArgs(outputDirPath)
			if err != nil {
				return fmt.Errorf("%v: could not write args to fs", err)
			}
		}

	}
	return nil
}

// Set label to Github pr if LABEL_NAME is set.
func (c commentMonitorClient) postLabel(ctx context.Context) error {
	if os.Getenv("LABEL_NAME") != "" {
		if err := c.ghClient.createLabel(ctx, os.Getenv("LABEL_NAME")); err != nil {
			return fmt.Errorf("%v : couldn't set label", err)
		}
		log.Println("label successfully set")
	}
	return nil
}

// writeArgs writes all arguments to the file system.
func (c commentMonitorClient) writeArgs(outputDirPath string) error {
	for filename, content := range c.allArgs {
		data := []byte(content)
		err := ioutil.WriteFile(filepath.Join(outputDirPath, filename), data, 0644)
		if err != nil {
			return fmt.Errorf("%v: could not write arg to filesystem", err)
		}
		log.Printf("file added: %v", filepath.Join(outputDirPath, filename))
	}
	return nil
}

func (c commentMonitorClient) generateAndPostComment(ctx context.Context) error {
	if c.commentTemplate != "" {
		// Add all env vars to allArgs.
		for _, e := range os.Environ() {
			tmp := strings.Split(e, "=")
			c.allArgs[tmp[0]] = tmp[1]
		}
		// Generate the comment template.
		var buf bytes.Buffer
		commentTemplate := template.Must(template.New("Comment").Parse(c.commentTemplate))
		if err := commentTemplate.Execute(&buf, c.allArgs); err != nil {
			return err
		}
		// Post the comment.
		if err := c.ghClient.postComment(ctx, buf.String()); err != nil {
			return fmt.Errorf("%v : couldn't post generated comment", err)
		}
		log.Println("comment successfully posted")
	}
	return nil
}
