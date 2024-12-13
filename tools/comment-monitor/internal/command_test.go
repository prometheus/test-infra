// Copyright 2024 The Prometheus Authors
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

package internal

import (
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

const (
	eventTypeStart   = "prombench_start"
	eventTypeRestart = "prombench_restart"
	eventTypeStop    = "prombench_stop"
)

func testCommand(eventType, release string) *Command {
	c := &Command{
		Prefix:           "/prombench",
		Args:             map[string]string{},
		EventType:        eventType,
		ShouldVerifyUser: true,
	}
	if release != "" {
		c.Args["RELEASE"] = release
	}
	return c
}

func helpCommand() *Command {
	return &Command{
		Prefix: "/prombench",
		Args:   map[string]string{},
	}
}

type parseCommandCase struct {
	comment                string
	expect                 *Command
	expectErrCommentPrefix string
}

func testParseCommand(t *testing.T, c *Config, cases []parseCommandCase) {
	t.Helper()

	for _, tcase := range cases {
		t.Run(tcase.comment, func(t *testing.T) {
			cmd, found, pErr := ParseCommand(c, tcase.comment)

			// Incorrect syntax cases.
			if tcase.expectErrCommentPrefix != "" {
				if found == true {
					t.Fatal("expected not found, got true")
				}
				if pErr == nil {
					t.Fatal("expected error, got nil and found=", found)
				}
				if !strings.HasPrefix(pErr.ToComment(), tcase.expectErrCommentPrefix) {
					t.Fatalf("Error comment does not match expected prefix:\n%s\n\ncomment:\n%s", tcase.expectErrCommentPrefix, pErr.ToComment())
				}
				return
			}

			if pErr != nil {
				t.Fatalf("expected no error, got %q", pErr)
			}

			// Triggering event cases.
			if tcase.expect != nil {
				if found == false {
					t.Fatal("expected found, got false")
				}
				if cmd == nil {
					t.Fatal("expected command, got nil")
				}
				// Don't test those fields for now.
				cmd.SuccessCommentTemplate = ""
				cmd.DebugCMDLine = ""
				cmd.SuccessLabel = ""
				if diff := cmp.Diff(*cmd, *tcase.expect); diff != "" {
					t.Fatalf("-expect vs +got: %v", diff)
				}
				return
			}

			// Not matching cases.
			if found == true {
				t.Fatal("expected not found, got true")
			}
		})
	}
}

func TestParseCommand(t *testing.T) {

	c, err := ParseConfig("./testconfig.yaml")
	if err != nil {
		t.Fatal(err)
	}
	testParseCommand(t, c, []parseCommandCase{
		{
			comment:                "/prombench",
			expectErrCommentPrefix: "Incorrect `/prombench` syntax; no matching command found.",
		},
		{
			comment: "/prombench v3.0.0",
			expect:  testCommand(eventTypeStart, "v3.0.0"),
		},
		{
			comment: "/prombench restart v3.0.0",
			expect:  testCommand(eventTypeRestart, "v3.0.0"),
		},
		{
			comment: "/prombench cancel",
			expect:  testCommand(eventTypeStop, ""),
		},
		{
			comment: "/prombench help",
			expect:  helpCommand(),
		},
		// Different versions based on the provided  args_regex: ^\s+(?P<RELEASE>master|main|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$
		{
			comment: "/prombench main",
			expect:  testCommand(eventTypeStart, "main"),
		},
		// Text at the end is generally accepted, after \n.
		{
			comment: "/prombench v3.0.0\n",
			expect:  testCommand(eventTypeStart, "v3.0.0"),
		},
		{
			comment: "/prombench v3.0.0\n\nYolo",
			expect:  testCommand(eventTypeStart, "v3.0.0"),
		},
		// Incorrect syntax cases.
		{
			comment:                "/prombench v3.0.0 garbage",
			expectErrCommentPrefix: "Incorrect `/prombench` syntax;  command requires at least one argument that matches `" + `(?P<RELEASE>master|main|v[0-9]+\.[0-9]+\.[0-9]+\S*)$` + "` regex.",
		},
		{
			comment:                "/prombench restart v3.0.0 garbage",
			expectErrCommentPrefix: "Incorrect `/prombench` syntax; restart command requires at least one argument that matches `" + `(?P<RELEASE>master|main|v[0-9]+\.[0-9]+\.[0-9]+\S*)$` + "` regex.",
		},
		{
			comment:                "/prombench restartv3.0.0 garbage",
			expectErrCommentPrefix: "Incorrect `/prombench` syntax;  command requires at least one argument that matches `" + `(?P<RELEASE>master|main|v[0-9]+\.[0-9]+\.[0-9]+\S*)$` + "` regex.",
		},
		{
			comment:                "/prombench cancel garbage",
			expectErrCommentPrefix: "Incorrect `/prombench` syntax; cancel command expects no arguments, but got some.",
		},
		{
			comment:                "/prombench not-a-version",
			expectErrCommentPrefix: "Incorrect `/prombench` syntax;  command requires at least one argument that matches `" + `(?P<RELEASE>master|main|v[0-9]+\.[0-9]+\.[0-9]+\S*)$` + "` regex.",
		},
		// Not matching cases.
		{comment: ""},
		{comment: "How to start prombench?\nyolo\nthanks"},
		// Space has to be used between prefix and command.
		{comment: "/prombenchv3.0.0"},
		{comment: "/prombenchv3.0.0 v3.0.0"},
		{comment: "/prombenchcancel"},
		// Text in the front is not matching prombench.
		// TODO(bwplotka): Consider accepting things before /<prefix
		{comment: "How to start prombench? I think it was something like /prombench main"},
		{comment: "How to start prombench? I think it was something like /prombench main\nor something"},
		{comment: "How to start prombench? I think it was something like:\n\n /prombench main"},
		{comment: "How to start prombench? I think it was something like:\n\n /prombench main\n"},
		{comment: "How to start prombench? I think it was something like:\n\n /prombench main\n\nYolo"},
	})
}

// NOTE(bwplotka): Simplified version of TestParseCommand that literally uses
// our production comment monitoring configuration in the same repo.
func TestParseCommand_ProdCommentMonitorConfig(t *testing.T) {
	const prodCommentMonitorConfigMap = "../../../prombench/manifests/cluster-infra/7a_commentmonitor_configmap_noparse.yaml"

	b, err := os.ReadFile(prodCommentMonitorConfigMap)
	if err != nil {
		t.Fatal(err, prodCommentMonitorConfigMap)
	}

	cfm := struct {
		Data struct {
			ConfigYaml string `yaml:"config.yml"`
		} `yaml:"data"`
	}{}
	if err := yaml.Unmarshal(b, &cfm); err != nil {
		t.Fatalf("parsing %v: %v", prodCommentMonitorConfigMap, err)
	}
	if len(cfm.Data.ConfigYaml) == 0 {
		t.Fatalf("expected commentMonitor production configuration in %v data.config.yml field, got nothing", prodCommentMonitorConfigMap)
	}

	c, err := parseConfigContent([]byte(cfm.Data.ConfigYaml))
	if err != nil {
		t.Fatal(err)
	}
	testParseCommand(t, c, []parseCommandCase{
		{
			comment:                "/prombench",
			expectErrCommentPrefix: "Incorrect `/prombench` syntax; no matching command found.",
		},
		{
			comment: "/prombench v3.0.0\nSome text after",
			expect:  testCommand(eventTypeStart, "v3.0.0"),
		},
		{
			comment: "/prombench main\nSome text after",
			expect:  testCommand(eventTypeStart, "main"),
		},
		{
			comment: "/prombench master\nSome text after",
			expect:  testCommand(eventTypeStart, "master"),
		},
		{
			comment: "/prombench restart v3.0.0\nSome text after",
			expect:  testCommand(eventTypeRestart, "v3.0.0"),
		},
		{
			comment: "/prombench cancel\nSome text after",
			expect:  testCommand(eventTypeStop, ""),
		},
		// Not matching cases.
		{comment: ""},
		{comment: "How to start prombench? I think it was something like:\n\n /prombench main\n\nYolo"},
	})
}
