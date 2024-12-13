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
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

// Config allows defining custom commands that with their
// arguments. Those commands will be then parsed from GH issue comments, if
// anyone will comment with `/<prefix> <command> <args>` line.
type Config struct {
	Prefixes []*PrefixConfig `yaml:"prefixes"`
}

func ParseConfig(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %v: %w", file, err)
	}
	return parseConfigContent(data)
}

func parseConfigContent(content []byte) (_ *Config, err error) {
	cfg := &Config{}
	if err := yaml.UnmarshalStrict(content, cfg); err != nil {
		return nil, fmt.Errorf("cannot unmarshal data: %w", err)
	}
	if len(cfg.Prefixes) == 0 {
		return nil, errors.New("empty configuration; no prefix")
	}
	for _, p := range cfg.Prefixes {
		if len(p.Commands) == 0 {
			return nil, fmt.Errorf("empty configuration; no command for /%v", p.Prefix)
		}
		for _, c := range p.Commands {
			if c.Name == "" && c.ArgsRegex == "" {
				return nil, fmt.Errorf("/%v bad config; default commands cannot have empty args_regex (no required arguments)", p.Prefix)
			}
			if strings.ToLower(c.Name) == "help" {
				return nil, fmt.Errorf("/%v bad config; 'help' command name is reserved", p.Prefix)

			}
			if strings.HasPrefix(c.ArgsRegex, "^") {
				return nil, fmt.Errorf("/%v bad config; args_regex has to be front open, got %v", p.Prefix, c.ArgsRegex)
			}

			c.argsRegex, err = regexp.Compile(c.ArgsRegex)
			if err != nil {
				return nil, fmt.Errorf("/%v bad config; command %v args_regex %v doesn't compile: %w", p.Prefix, c.Name, c.ArgsRegex, err)
			}

			commandArgsNames := c.argsRegex.SubexpNames()[1:]
			for _, argName := range commandArgsNames {
				if argName == "" {
					return nil, fmt.Errorf("/%v bad config; command %v named groups in regex are mandatory; got %v", p.Prefix, c.Name, c.ArgsRegex)
				}
			}
		}
	}
	return cfg, nil
}

type PrefixConfig struct {
	Prefix     string           `yaml:"prefix"`
	Help       string           `yaml:"help"`
	VerifyUser bool             `yaml:"verify_user"`
	Commands   []*CommandConfig `yaml:"commands"`
}

type CommandConfig struct {
	Name            string `yaml:"name"`
	EventType       string `yaml:"event_type"`
	CommentTemplate string `yaml:"comment_template"`
	ArgsRegex       string `yaml:"args_regex"`
	Label           string `yaml:"label"`

	argsRegex *regexp.Regexp
}

// Command represents a read-only, parsed command to dispatch with
// additional details for the GH comment update.
type Command struct {
	Prefix    string
	EventType string
	Args      map[string]string

	ShouldVerifyUser       bool
	SuccessCommentTemplate string
	SuccessLabel           string

	DebugCMDLine string
}

type CommandParseError struct {
	error
	help string
}

func (c *CommandParseError) ToComment() string {
	return c.help
}

func hasExactPrefix(s, token string) bool {
	return s == token || strings.HasPrefix(s, token+" ") || strings.HasPrefix(s, token+"\n")
}

// ParseCommand parses command to dispatch from the issue comment, given the provided configuration.
func ParseCommand(cfg *Config, comment string) (_ *Command, ok bool, err *CommandParseError) {
	comment = strings.TrimSpace(comment)

	// TODO(bwplotka): Consider accepting things before /<prefix

	// Find the prefix.
	var prefix *PrefixConfig
	for _, p := range cfg.Prefixes {
		if hasExactPrefix(comment, p.Prefix) {
			prefix = p
			break
		}
	}
	if prefix == nil {
		return nil, false, nil
	}

	// Our command line is a single line from the prefix (including it) to a new line.
	i := strings.Index(comment, "\n")
	if i == -1 {
		i = len(comment)
	}
	cmdLine := comment[:i]
	rest := cmdLine[len(prefix.Prefix):]

	// Is it help?
	if hasExactPrefix(rest, " help") {
		return &Command{
			Args:                   map[string]string{},
			Prefix:                 prefix.Prefix,
			SuccessCommentTemplate: prefix.Help,
			DebugCMDLine:           cmdLine,
		}, true, nil
	}

	// Find the command.
	var cmdConfig *CommandConfig
	var defaultCmdConfig *CommandConfig
	for _, c := range prefix.Commands {
		if c.Name == "" {
			defaultCmdConfig = c
			continue
		}

		if hasExactPrefix(rest, " "+c.Name) {
			cmdConfig = c
			break
		}
	}

	if cmdConfig == nil {
		// No explicit command found. Is it a default command? (they have to have arguments)
		if defaultCmdConfig != nil && strings.HasPrefix(rest, " ") {
			cmdConfig = defaultCmdConfig
		} else {
			return nil, false, &CommandParseError{
				error: fmt.Errorf("no matching command found for comment line: %v", cmdLine),
				help:  fmt.Sprintf("Incorrect `%v` syntax; no matching command found.\n\n%s", prefix.Prefix, prefix.Help),
			}
		}
	}

	cmd := &Command{
		Args:      map[string]string{},
		Prefix:    prefix.Prefix,
		EventType: cmdConfig.EventType,

		ShouldVerifyUser:       prefix.VerifyUser,
		SuccessCommentTemplate: cmdConfig.CommentTemplate,
		SuccessLabel:           cmdConfig.Label,
		DebugCMDLine:           cmdLine,
	}
	if len(cmdConfig.Name) > 0 {
		rest = rest[len(cmdConfig.Name)+1:] // plus prefixed space.
	}

	if cmdConfig.ArgsRegex == "" {
		// Ensure there are no more characters.
		if len(rest) > 0 {
			return nil, false, &CommandParseError{
				error: fmt.Errorf("command expected no argument, but got some '%v' for cmdLine: '%v'", rest, cmdLine),
				help:  fmt.Sprintf("Incorrect `%v` syntax; %v command expects no arguments, but got some.\n\n%s", prefix.Prefix, cmdConfig.Name, prefix.Help),
			}
		}
		return cmd, true, nil
	}
	// Parse required arguments.
	if !cmdConfig.argsRegex.MatchString(rest) {
		return nil, false, &CommandParseError{
			error: fmt.Errorf("command requires at least one argument, matching '%v' regex on '%v' string for cmdLine '%v'", cmdConfig.ArgsRegex, rest, cmdLine),
			help:  fmt.Sprintf("Incorrect `%v` syntax; %v command requires at least one argument that matches `%v` regex.\n\n%s", prefix.Prefix, cmdConfig.Name, cmdConfig.ArgsRegex, prefix.Help),
		}
	}

	args := cmdConfig.argsRegex.FindStringSubmatch(rest)[1:]
	commandArgsNames := cmdConfig.argsRegex.SubexpNames()[1:]
	for i, argName := range commandArgsNames {
		if argName == "" {
			return nil, false, &CommandParseError{
				error: fmt.Errorf("named groups in regex are mandatory; should be validated on config read, got %v", cmdConfig.ArgsRegex),
			}
		}
		cmd.Args[argName] = args[i]
	}
	return cmd, true, nil
}
