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
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"

	"github.com/google/go-github/v29/github"
	"github.com/nelkinda/health-go"
	"github.com/oklog/run"
	"golang.org/x/sync/singleflight"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/prometheus/test-infra/tools/comment-monitor/internal"
)

func main() {
	var configFile, whSecretFile, listenPort, logLevelStr string

	app := kingpin.New(filepath.Base(os.Args[0]), "comment-monitor: A GH webhook "+
		"that watches GH Issue (and PR) comments for '/<prefix> [<command>] [<version>] [<flags>]'"+
		"commands. On each command, it dispatches appropriate command as an event to GitHub repository API and notifies the same issue/PR.")
	app.HelpFlag.Short('h')
	app.Flag("config", "Path to the config file.").
		Default("./config.yml").
		StringVar(&configFile)
	app.Flag("whsecret", "Path to the webhook secret file for the payload signature validation.").
		Default("./whsecret").
		StringVar(&whSecretFile)
	app.Flag("port", "Port number to run webhook on.").
		Default("8080").
		StringVar(&listenPort)
	app.Flag("log.level", "Logging level, available values: 'debug', 'info', 'warn', 'error'.").
		Default("info").
		StringVar(&logLevelStr)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	var logLevel slog.Level
	if err := logLevel.UnmarshalText([]byte(logLevelStr)); err != nil {
		log.Fatal("failed to parse -log.level flag", err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

	d := newDispatcher(logger, configFile, whSecretFile)
	mux := http.NewServeMux()
	mux.HandleFunc("/", d.HandleIssue) // Issue means both GitHub Issue and PR.

	healthHandler := health.New(health.Health{}).Handler
	mux.HandleFunc("/-/health", healthHandler)
	mux.HandleFunc("/-/ready", healthHandler)

	var g run.Group
	{
		addr := fmt.Sprintf(":%v", listenPort)
		httpSrv := &http.Server{Addr: addr, Handler: mux}

		g.Add(func() error {
			logger.Info("server is ready to handle requests", "address", addr)
			return httpSrv.ListenAndServe()
		}, func(_ error) {
			_ = httpSrv.Shutdown(context.Background())
		})
	}
	g.Add(run.SignalHandler(context.Background(), os.Interrupt, syscall.SIGTERM))
	if err := g.Run(); err != nil {
		logger.Error("running comment-monitor failed", "err", err)
		os.Exit(1)
	}
	logger.Info("sink finished")
}

type dispatcher struct {
	logger                   *slog.Logger
	configFile, whSecretFile string

	sfg singleflight.Group
}

func newDispatcher(logger *slog.Logger, configFile, whSecretFile string) *dispatcher {
	return &dispatcher{logger: logger, configFile: configFile, whSecretFile: whSecretFile}
}

func (d *dispatcher) readConfigAndSecrets() (*internal.Config, []byte, string, error) {
	type result struct {
		cfg      *internal.Config
		whSecret []byte
		ghToken  string
	}

	// Do it under a singleflight, as there's no
	// need for concurrent requests to re-read files in the same time.
	res, err, _ := d.sfg.Do("config-and-secret", func() (_ any, err error) {
		r := result{}

		// Get a fresh config.
		r.cfg, err = internal.ParseConfig(d.configFile)
		if err != nil {
			return nil, err
		}

		// Get a fresh webhook secret.
		r.whSecret, err = os.ReadFile(d.whSecretFile)
		if err != nil {
			return nil, err
		}

		// Get a fresh GH token.
		r.ghToken = os.Getenv("GITHUB_TOKEN")
		if r.ghToken == "" {
			return nil, fmt.Errorf("GITHUB_TOKEN env var missing")
		}
		return r, nil
	})
	return res.(result).cfg, res.(result).whSecret, res.(result).ghToken, err
}

func handleErr(w http.ResponseWriter, logger *slog.Logger, errMsg string, statusCode int, err error) {
	logger.With("err", err, "code", statusCode).Error(errMsg)
	http.Error(w, errMsg, statusCode)
}

func (d *dispatcher) HandleIssue(w http.ResponseWriter, r *http.Request) {
	defer func() {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}()

	logger := d.logger
	reqID := r.Header.Get("X-GitHub-Delivery")
	if reqID != "" {
		logger = logger.With("reqID", reqID)
	}

	// Get fresh a configuration and secret on every request.
	cfg, whSecret, ghToken, err := d.readConfigAndSecrets()
	if err != nil {
		handleErr(w, logger, "configuration or secrets are incorrect", http.StatusInternalServerError, err)
		return
	}

	// Validate payload, including its signature using the secret.
	payload, err := github.ValidatePayload(r, whSecret)
	if err != nil {
		handleErr(w, logger, "failed to validate webhook payload", http.StatusBadRequest, err)
		return
	}

	// Parse webhook event.
	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		handleErr(w, logger, "failed to parse GH event payload", http.StatusBadRequest, err)
		return
	}

	e, ok := event.(*github.IssueCommentEvent)
	if !ok {
		logger := logger.With("eventType", fmt.Sprintf("%T", event))
		handleErr(w, logger, "only issue_comment event is supported", http.StatusBadRequest, err)
		return
	}

	if *e.Action != "created" {
		logger.Debug("issue_comment type must be 'created', updates/edits are not supported", "action", *e.Action)
		http.Error(w, "issue_comment type must be 'created'", http.StatusOK) // Using http.Error as a nice text response util.
		return
	}

	eventDetails := internal.NewEventDetails(e)
	ghClient, err := internal.NewGithubClient(r.Context(), ghToken, eventDetails)
	if err != nil {
		handleErr(w, logger, "could not create GitHub client against the given repository", http.StatusBadRequest, err)
		return
	}

	logger = logger.With("repo", eventDetails.Repo, "issue", eventDetails.PR, "author", eventDetails.Author)

	cmd, found, parseErr := internal.ParseCommand(cfg, e.GetComment().GetBody())
	if parseErr != nil {
		if comment := parseErr.ToComment(); comment != "" {
			if postErr := ghClient.PostComment(comment); postErr != nil {
				logger := logger.With("postErr", postErr)
				handleErr(w, logger, "could not post comment to GitHub on failed command parsing", http.StatusBadRequest, parseErr)
				return
			}
		}
		// TODO(bwplotka) Post comment about this failure?
		handleErr(w, logger, "parsing command from comment failed", http.StatusBadRequest, err)
		return
	}

	if !found {
		logger.Debug("issue does not contain any command")
		http.Error(w, "not a command", http.StatusOK) // Using http.Error as a nice text response util.
		return
	}

	// Verify user if configured.
	if cmd.ShouldVerifyUser {
		var allowed bool
		allowedAssociations := []string{"COLLABORATOR", "MEMBER", "OWNER"}
		for _, a := range allowedAssociations {
			if a == eventDetails.AuthorAssociation {
				allowed = true
			}
		}
		if !allowed {
			b := fmt.Sprintf("@%s is not a org member nor a collaborator and cannot execute benchmarks.", eventDetails.Author)
			logger := logger.With("allowed", strings.Join(allowedAssociations, ","))
			if err := ghClient.PostComment(b); err != nil {
				handleErr(w, logger, "user not allowed to run command; also could not post comment to GitHub", http.StatusForbidden, err)
			} else {
				handleErr(w, logger, "user not allowed to run command", http.StatusForbidden, nil)
			}
			return
		}

		logger = logger.With("cmdLine", cmd.DebugCMDLine)
		logger.Info("dispatching a new command and updating issue")

		// Combine all arguments for both dispatch and the comment update.
		allArgs := cmd.Args
		allArgs["PR_NUMBER"] = strconv.Itoa(eventDetails.PR)
		allArgs["LAST_COMMIT_SHA"], err = ghClient.GetLastCommitSHA()
		if err != nil {
			// TODO(bwplotka) Post comment about this failure?
			handleErr(w, logger, "could not fetch SHA, which likely means it's an issue, not a pull request. Non-PRs are not supported.", http.StatusBadRequest, err)
			return
		}

		logger = logger.With("evenType", cmd.EventType, "args", fmt.Sprintf("%v", allArgs))
		if err = ghClient.Dispatch(cmd.EventType, allArgs); err != nil {
			// TODO(bwplotka) Post comment about this failure?
			handleErr(w, logger, "could not dispatch", http.StatusInternalServerError, err)
			return
		}
		logger.Info("dispatched repository GitHub payload")

		// Update the issue.
		comment, err := executeCommentTemplate(cmd.SuccessCommentTemplate, allArgs)
		if err != nil {
			handleErr(w, logger, "failed to execute template", http.StatusInternalServerError, err)
			return
		}

		if err = ghClient.PostComment(comment); err != nil {
			handleErr(w, logger, "dispatch successful; but could not post comment to GitHub", http.StatusInternalServerError, err)
			return
		}

		if cmd.SuccessLabel != "" {
			if err = ghClient.PostLabel(cmd.SuccessLabel); err != nil {
				handleErr(w, logger, "dispatch successful; but could not post label to GitHub", http.StatusInternalServerError, err)
				return
			}
		}
	}
}

func executeCommentTemplate(commentTemplate string, args map[string]string) (string, error) {
	argsCpy := make(map[string]string, len(args)) // TODO(bwplotka): Looks unsafe, we might want to type the known options.
	if len(args) > 0 {
		for k, v := range args {
			argsCpy[k] = v
		}
	}
	for _, e := range os.Environ() {
		tmp := strings.Split(e, "=")
		argsCpy[tmp[0]] = tmp[1]
	}

	// Generate the comment template.
	var buf bytes.Buffer
	ct := template.Must(template.New("Comment").Parse(commentTemplate))
	if err := ct.Execute(&buf, argsCpy); err != nil {
		return "", fmt.Errorf("templating failed: %w", err)
	}
	return buf.String(), nil
}
