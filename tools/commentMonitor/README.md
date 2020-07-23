# commentMonitor
Simple webhook server to parse GitHub comments and take actions based on the comment.

Currently it only works with [`issue_comment` event](https://developer.github.com/v3/activity/events/types/#issuecommentevent) coming from PRs.

### Environment Variables:
- `GITHUB_TOKEN` : GitHub oauth token used for posting comments and settings the label.
- Any other environment variable used in any of the comment templates in `config.yml`.

### Setting up the webhook server
To specify the config file for commentMonitor use the the `--config` flag.

Example `config.yml` file:
```yaml
prefixes:
  - prefix: /prombench
    help_template: |
      Get prombench syntax help here.
  - prefix: /funcbench
    help_template: |
      Get funcbench syntax help [here](https://canbealink).
eventmaps:
  - event_type: prombench_stop
    regex_string: (?mi)^/prombench\s+cancel\s*$
    comment_template: |
      Benchmark cancel is in progress.
    label: prombench
```

Before comments are matched against `regex_string`, they are checked if they start with any of the prefixes mentioned in `prefixes`. If not, the request is simply dropped.  Once a comment matches with `regex_string`, commentMonitor will trigger a [`repository_dispatch` event](https://developer.github.com/v3/repos/#create-a-repository-dispatch-event) with the event type `event_type` and then post a comment to the issue/pr with `comment_template`. The extracted out arguments will be passed to the [`client_payload`](https://developer.github.com/v3/repos/#example-5) of the `repository_dispatch` event.

If the matching with `regex_string` fails, then a comment with the `help_template` for that prefix is posted back to the corresponding issue/pr.

### Setting up the GitHub webhook
- Create a personal access token with the scope `public_repo` and `write:discussion` and set the environment variable `GITHUB_TOKEN` with it.
- Set the webhook server URL as the webhook URL in the repository settings and set the content type to `application/json`.

## Extracting arguments
The `regex_string` provided in `config.yml` is used to parse the comment into separate arguments. Additionally, some internal args are automatically set, eg. `PR_NUMBER` and `LAST_COMMMIT_SHA`.

If `regex_string` contains a capturing groups, using [named groups](https://godoc.org/regexp/syntax) is mandatory so that each comment argument is named after the regex group.

For example, the following regex will create an argument named `RELEASE` with the content of the capture group:
```
(?mi)^/prombench\s*(?P<RELEASE>master|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$
```

#### Usage and examples:
[embedmd]:# (commentMonitor-flags.txt)
```txt
usage: commentMonitor [<flags>]

commentMonitor GithubAction - Post and monitor GitHub comments.

Flags:
  -h, --help                   Show context-sensitive help (also try --help-long
                               and --help-man).
      --webhooksecretfile="./whsecret"
                               path to webhook secret file
      --no-verify-user         disable verifying user
      --config="./config.yml"  Filepath to config file.
      --port="8080"            port number to run webhook in.

```
### Building Docker Image

From the repository root:

```
docker build -t prominfra/comment-monitor:master .
```
