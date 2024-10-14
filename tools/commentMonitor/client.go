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
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"text/template"
)

type commentMonitorClient struct {
	ghClient         *githubClient
	allArgs          map[string]string
	regex            *regexp.Regexp
	events           []webhookEvent
	prefixes         []commandPrefix
	helpTemplate     string
	shouldVerifyUser bool
	eventType        string
	commentTemplate  string
	label            string
}

// Set eventType and commentTemplate if
// regexString is validated against provided command.
func (c *commentMonitorClient) validateRegex(command string) bool {
	for _, e := range c.events {
		c.regex = regexp.MustCompile(e.RegexString)
		if c.regex.MatchString(command) {
			c.commentTemplate = e.CommentTemplate
			c.eventType = e.EventType
			c.label = e.Label
			log.Println("comment validation successful")
			return true
		}
	}
	return false
}

func (c *commentMonitorClient) checkCommandPrefix(command string) bool {
	for _, p := range c.prefixes {
		if strings.HasPrefix(command, p.Prefix) {
			c.helpTemplate = p.HelpTemplate
			c.shouldVerifyUser = p.VerifyUser
			return true
		}
	}
	return false
}

// Verify if user is allowed to perform activity.
func (c commentMonitorClient) verifyUser() error {
	if c.shouldVerifyUser {
		var allowed bool
		allowedAssociations := []string{"COLLABORATOR", "MEMBER", "OWNER"}
		for _, a := range allowedAssociations {
			if a == c.ghClient.authorAssociation {
				allowed = true
			}
		}
		if !allowed {
			b := fmt.Sprintf("@%s is not a org member nor a collaborator and cannot execute benchmarks.", c.ghClient.author)
			if err := c.ghClient.postComment(b); err != nil {
				return fmt.Errorf("%w : couldn't post comment", err)
			}
			return fmt.Errorf("author is not a member or collaborator")
		}
		log.Println("author is a member or collaborator")
	}
	return nil
}

// Extract args if regexString provided.
func (c *commentMonitorClient) extractArgs(command string) error {
	var err error
	if c.regex != nil {
		// Add command arguments.
		commandArgs := c.regex.FindStringSubmatch(command)[1:]
		commandArgsNames := c.regex.SubexpNames()[1:]
		for i, argName := range commandArgsNames {
			if argName == "" {
				return fmt.Errorf("using named groups is mandatory")
			}
			c.allArgs[argName] = commandArgs[i]
		}

		// Add non-comment arguments if any.
		c.allArgs["PR_NUMBER"] = strconv.Itoa(c.ghClient.pr)
		c.allArgs["LAST_COMMIT_SHA"], err = c.ghClient.getLastCommitSHA()
		if err != nil {
			return fmt.Errorf("%w: could not fetch SHA", err)
		}

		// TODO (geekodour) : We could run this in a separate method.
		err = c.ghClient.createRepositoryDispatch(c.eventType, c.allArgs)
		if err != nil {
			return fmt.Errorf("%w: could not create repository_dispatch event", err)
		}
	}
	return nil
}

func (c commentMonitorClient) postLabel() error {
	if c.label != "" {
		if err := c.ghClient.createLabel(c.label); err != nil {
			return fmt.Errorf("%w : couldn't set label", err)
		}
		log.Println("label successfully set")
	}
	return nil
}

func (c commentMonitorClient) generateAndPostSuccessComment() error {
	return c.generateAndPostComment(c.commentTemplate)
}

func (c commentMonitorClient) generateAndPostErrorComment() error {
	return c.generateAndPostComment(c.helpTemplate)
}

func (c commentMonitorClient) generateAndPostComment(commentTemplate string) error {
	if commentTemplate != "" {
		// Add all env vars to allArgs.
		for _, e := range os.Environ() {
			tmp := strings.Split(e, "=")
			c.allArgs[tmp[0]] = tmp[1]
		}
		// Generate the comment template.
		var buf bytes.Buffer
		ct := template.Must(template.New("Comment").Parse(commentTemplate))
		if err := ct.Execute(&buf, c.allArgs); err != nil {
			return err
		}
		// Post the comment.
		if err := c.ghClient.postComment(buf.String()); err != nil {
			return fmt.Errorf("%w : couldn't post generated comment", err)
		}
		log.Println("comment successfully posted")
	}
	return nil
}
