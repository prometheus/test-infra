# commentMonitor
Simple webhook server to parse GitHub comments and take actions based on the comment.

It can be run both as a webhook and a GitHub Action, when running it as a GitHub action it can only post a github comment.

## Running as a webhook

### Environment Variables:
- `LABEL_NAME`: If set, will add the label to the PR.
- `GITHUB_TOKEN` : GitHub oauth token used for posting comments and settings the label.
- Any other environment variable used in any of the comment templates in `eventmap.yml`.

### Setting up the webhook server
Running comment monitor with the `--webhook` flag starts it in the webhook mode, it also requires the eventmap file which is specified by the `--eventmap` flag. It currently only supports `issue_comment` GitHub events.

The `regex_string`, `event_type` and `comment_template` can be specified in the `eventmap.yml` file.

Example content of the `eventmap.yml` file:
```
- event_type: prombench_stop
  regex_string: (?mi)^/prombench\s+cancel\s*$
  comment_template: |
    Benchmark cancel is in progress.
```

If a GitHub comment matches with `regex_string`, then commentMonitor will trigger a [`repository_dispatch`](https://developer.github.com/v3/repos/#create-a-repository-dispatch-event) with the event type `event_type` and then post a comment to the issue with `comment_template`. The extracted out arguments will be passed to the [`client_payload`](https://developer.github.com/v3/repos/#example-5) of the `repository_dispatch` event.



### Setting up the GitHub webhook
- Create a personal access token with the scope `public_repo` and `write:discussion` and set the environment variable `GITHUB_TOKEN` with it.
- Set the webhook server URL as the webhook URL in the repository settings.

## Running as a GitHub Action
commentMonitor can also be run as a Github Action to post comment using Golang templating.

### Environment Variables:
- `COMMENT_TEMPLATE` : A comment template using Golang template variables substitutions. If content text includes a variable name `{{ index . "SOME_VAR" }}` that exists as an env variable or comment argument it is expanded with the content of the variable.
- `GITHUB_TOKEN`
- `GITHUB_ORG`
- `GITHUB_REPO`
- `PR_NUMBER`

## Extracting arguments
The `regex_string` provided in `eventmap.yml` is used to parse the comment into separate arguments. Additionally, some internal args are automatically set, eg. `PR_NUMBER`.

Using [regex named groups](https://godoc.org/regexp/syntax) is mandatory so that each comment argument is named after the regex group.

For example, the following regex will create an argument named `RELEASE` with the content of the capture group:
```
(?mi)^/prombench\s*(?P<RELEASE>master|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$
```

### Docker image build
From the repository root:
```
$ make docker DOCKERFILE_PATH=tools/commentMonitor/Dockerfile DOCKER_IMAGE_NAME=comment-monitor DOCKER_IMAGE_TAG=0.0.1
```

#### Usage and examples:
```
./commentMonitor --help
```
