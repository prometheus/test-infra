# commentMonitor

A simple webhook server designed to parse GitHub comments and execute actions based on the comment content. Currently, it only supports the [`issue_comment` event](https://developer.github.com/v3/activity/events/types/#issuecommentevent) triggered by pull requests (PRs).

## Table of Contents

1. [Environment Variables](#environment-variables)
2. [Setting Up the Webhook Server](#setting-up-the-webhook-server)
   - [Example `config.yml` File](#example-configyml-file)
3. [Setting Up the GitHub Webhook](#setting-up-the-github-webhook)
4. [Extracting Arguments](#extracting-arguments)
5. [Usage and Examples](#usage-and-examples)
6. [Building Docker Image](#building-docker-image)

## Environment Variables

- `GITHUB_TOKEN`: GitHub OAuth token used for posting comments and setting labels.
- Any other environment variable used within the comment templates in `config.yml`.

## Setting Up the Webhook Server

To specify the configuration file for `commentMonitor`, use the `--config` flag.

### Example `config.yml` File

```yaml
prefixes:
  - prefix: /prombench
    help_template: |
      Get prombench syntax help here.

eventmaps:
  - event_type: prombench_stop
    regex_string: (?mi)^/prombench\s+cancel\s*$
    comment_template: |
      Benchmark cancel is in progress.
    label: prombench
```

**How It Works:**
- Comments are first checked to see if they start with any of the prefixes specified in `prefixes`. If not, the request is dropped.
- If the prefix is matched, but the subsequent content does not match the `regex_string`, a comment with the `help_template` for that prefix is posted back to the issue/PR.
- If a comment matches the `regex_string`, `commentMonitor` will trigger a [`repository_dispatch` event](https://developer.github.com/v3/repos/#create-a-repository-dispatch-event) with the specified `event_type`.
- A comment will also be posted to the issue/PR with the `comment_template`.
- Any arguments extracted by the `regex_string` will be passed to the [`client_payload`](https://developer.github.com/v3/repos/#example-5) of the `repository_dispatch` event.

## Setting Up the GitHub Webhook

1. **Create a Personal Access Token**:
   - Generate a personal access token with the scopes `public_repo` and `write:discussion`.
   - Set the environment variable `GITHUB_TOKEN` to this token.

2. **Configure the Webhook**:
   - Set the webhook server URL as the webhook URL in the repository settings.
   - Set the content type to `application/json`.

## Extracting Arguments

- The `regex_string` in `config.yml` is used to parse comments into separate arguments.
- Some internal arguments are automatically set, such as `PR_NUMBER` and `LAST_COMMIT_SHA`.
- If the `regex_string` includes capturing groups, you must use [named groups](https://godoc.org/regexp/syntax) so that each comment argument is named after the corresponding regex group.

**Example**:

The following regex creates an argument named `RELEASE`:

```regex
(?mi)^/prombench\s*(?P<RELEASE>master|main|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$
```

## Usage and Examples

```txt
usage: commentMonitor [<flags>]

commentMonitor GitHub Action - Post and monitor GitHub comments.

Flags:
  -h, --help                   Show context-sensitive help (also try --help-long and --help-man).
      --webhooksecretfile="./whsecret"   Path to webhook secret file.
      --config="./config.yml"  Filepath to config file.
      --port="8080"            Port number to run webhook on.
```

## Building Docker Image

To build the Docker image for `commentMonitor`:

```bash
docker build -t prominfra/comment-monitor:master .
```
