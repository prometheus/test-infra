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
			if c.Name == "" && c.ArgName == "" && c.ArgRegex == "" {
				return nil, fmt.Errorf("/%v bad config; default commands cannot have empty arg_name and args_regex (no required arguments)", p.Prefix)
			}
			if strings.ToLower(c.Name) == "help" {
				return nil, fmt.Errorf("/%v bad config; 'help' command name is reserved", p.Prefix)
			}
			if c.ArgRegex != "" {
				if c.ArgName == "" {
					return nil, fmt.Errorf("/%v bad config; command '%v' arg_name cannot be empty, when arg_regex is specified", p.Prefix, c.Name)
				}
				c.argsRegex, err = regexp.Compile(c.ArgRegex)
				if err != nil {
					return nil, fmt.Errorf("/%v bad config; command %v args_regex %v doesn't compile: %w", p.Prefix, c.Name, c.ArgRegex, err)
				}
			} else {
				c.argsRegex = regexp.MustCompile(".*")
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
	Name            string            `yaml:"name"`
	EventType       string            `yaml:"event_type"`
	CommentTemplate string            `yaml:"comment_template"`
	ArgRegex        string            `yaml:"arg_regex"`
	ArgName         string            `yaml:"arg_name"`
	FlagArgs        map[string]string `yaml:"flag_args"` // flagName => argName
	Label           string            `yaml:"label"`

	argsRegex *regexp.Regexp
}

// Command represents a read-only, parsed command to dispatch with
// additional details for the GH comment update.
type Command struct {
	Prefix    string
	EventType string
	Args      map[string]string
	Flags     map[string]string

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

	// TODO(bwplotka): Consider accepting things before /<prefix, but be careful
	// on recursive flows (parsing own help comments).

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
	rest := strings.Split(strings.TrimSpace(cmdLine[len(prefix.Prefix):]), " ")
	if len(rest) == 0 || rest[0] == "" {
		return nil, false, &CommandParseError{
			error: fmt.Errorf("no matching command found for comment line: %v", cmdLine),
			help:  fmt.Sprintf("Incorrect `%v` syntax; no matching command found.\n\n%s", prefix.Prefix, prefix.Help),
		}
	}

	if rest[0] == "help" {
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

		if c.Name == rest[0] {
			cmdConfig = c
			break
		}
	}

	if cmdConfig == nil {
		if defaultCmdConfig == nil {
			return nil, false, &CommandParseError{
				error: fmt.Errorf("no matching command found for comment line: %v", cmdLine),
				help:  fmt.Sprintf("Incorrect `%v` syntax; no matching command found.\n\n%s", prefix.Prefix, prefix.Help),
			}
		}
		cmdConfig = defaultCmdConfig
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
		rest = rest[1:]
	}

	// We expect next token to be the required argument (if defined in config).
	if cmdConfig.ArgName == "" {
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "--") {
			return nil, false, &CommandParseError{
				error: fmt.Errorf("command expected no argument, but got some '%v' for cmdLine: '%v'", rest, cmdLine),
				help:  fmt.Sprintf("Incorrect `%v` syntax; %v command expects no arguments, but got some.\n\n%s", prefix.Prefix, cmdConfig.Name, prefix.Help),
			}
		}
	} else {
		// Check the required argument.
		if len(rest) == 0 || !cmdConfig.argsRegex.MatchString(rest[0]) {
			return nil, false, &CommandParseError{
				error: fmt.Errorf("command requires one argument, matching '%v' regex; got cmdLine '%v' and args %v", cmdConfig.argsRegex.String(), cmdLine, rest),
				help:  fmt.Sprintf("Incorrect `%v` syntax; %v command requires one argument that matches `%v` regex.\n\n%s", prefix.Prefix, cmdConfig.Name, cmdConfig.argsRegex.String(), prefix.Help),
			}
		}
		cmd.Args[cmdConfig.ArgName] = rest[0]
		rest = rest[1:]
	}

	// We expect only flags now.
	if len(cmdConfig.FlagArgs) > 0 {
		if err := parseFlags(rest, cmdConfig, cmd); err != nil {
			return nil, false, &CommandParseError{
				error: fmt.Errorf("command flag parsing failed for cmdLine '%v' and flags %v: %w", cmdLine, rest, err),
				help:  fmt.Sprintf("Incorrect `%v` syntax; %v command flag parsing failed: %v.\n\n%s", prefix.Prefix, cmdConfig.Name, err.Error(), prefix.Help),
			}
		}
	} else if len(rest) > 0 {
		return nil, false, &CommandParseError{
			error: fmt.Errorf("command does not expect any flags; got cmdLine '%v' and flags %v", cmdLine, rest),
			help:  fmt.Sprintf("Incorrect `%v` syntax; %v command expects no flags but got some.\n\n%s", prefix.Prefix, cmdConfig.Name, prefix.Help),
		}
	}
	return cmd, true, nil
}

func parseFlags(rest []string, cfg *CommandConfig, cmd *Command) error {
	// TODO(bwplotka: Naive flag parsing, make it support quoting, spaces etc later.
	for _, flag := range rest {
		if !strings.HasPrefix(flag, "--") {
			return fmt.Errorf("expected flag (starting with --), got %v", flag)
		}
		parts := strings.Split(flag, "=")
		if len(parts) != 2 {
			return fmt.Errorf("expected flag format '--<flag>=<value>', got %v", flag)
		}

		argName, ok := cfg.FlagArgs[strings.TrimPrefix(parts[0], "--")]
		if !ok {
			return fmt.Errorf("flag %v is not supported", flag)
		}
		cmd.Args[argName] = parts[1]
	}
	return nil
}
