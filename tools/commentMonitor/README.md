# commentMonitor
Simple webhook server to parse GitHub comments and take actions based on the comment.

Currently it only works with [`issue_comment` event](https://developer.github.com/v3/activity/events/types/#issuecommentevent) coming from PRs.

### Environment Variables:
- `LABEL_NAME`: If set, will add the label to the PR.
- `COMMAND_PREFIXES`: Comma separated list of command prefixes.
- `GITHUB_TOKEN` : GitHub oauth token used for posting comments and settings the label.
- Any other environment variable used in any of the comment templates in `eventmap.yml`.

### Setting up the webhook server
Running commentMonitor requires the eventmap file which is specified by the `--eventmap` flag.

The `regex_string`, `event_type` and `comment_template` can be specified in the `eventmap.yml` file.

Example content of the `eventmap.yml` file:
```yaml
- event_type: prombench_stop
  regex_string: (?mi)^/prombench\s+cancel\s*$
  comment_template: |
    Benchmark cancel is in progress.
```
```shell
$ export COMMENT_PREFIX='/funcbench,/prombench'
```

If a GitHub comment matches with `regex_string`, then commentMonitor will trigger a [`repository_dispatch`](https://developer.github.com/v3/repos/#create-a-repository-dispatch-event) with the event type `event_type` and then post a comment to the issue with `comment_template`. The extracted out arguments will be passed to the [`client_payload`](https://developer.github.com/v3/repos/#example-5) of the `repository_dispatch` event.


### Setting up the GitHub webhook
- Create a personal access token with the scope `public_repo` and `write:discussion` and set the environment variable `GITHUB_TOKEN` with it.
- Set the webhook server URL as the webhook URL in the repository settings and set the content type to `application/json`.

## Extracting arguments
The `regex_string` provided in `eventmap.yml` is used to parse the comment into separate arguments. Additionally, some internal args are automatically set, eg. `PR_NUMBER` and `LAST_COMMMIT_SHA`.

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
  -h, --help            Show context-sensitive help (also try --help-long and
                        --help-man).
      --webhooksecretfile="./whsecret"
                        path to webhook secret file
      --no-verify-user  disable verifying user
      --eventmap="./eventmap.yml"
                        Filepath to eventmap file.
      --port="8080"     port number to run webhook in.

```
### Building Docker Image

From the repository root:

```
docker build -t prominfra/comment-monitor:master .
```
