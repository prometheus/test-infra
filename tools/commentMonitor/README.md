# commentMonitor - Inspect github comments to extract arguments from them

`commentMonitor` expects a github event payload as the input and based on the
regex argument it extracts arguments out of the comment. It can can also post
comments and set labels on the pr from which the comment was received.

See [the github issue events api](https://developer.github.com/v3/issues/events/) for some examples.

If running as a webhook, the `regex_string`, `event_type` and `comment_template` can be specified in the `eventmap.yml` file. See [Running as a webhook for more information](#running-as-a-webhook)

#### Environment Variables:
- `COMMENT_TEMPLATE` (only used when **not** running in webhook mode) : If set, will post a comment with the content. It uses the Golang template variables substitutions. If content text includes a variable name `{{ index . "SOME_VAR" }}` that exists as an env variable or comment argument it is expanded with the content of the variable.
- `LABEL_NAME`: If set, will add the label to the PR.
- `GITHUB_TOKEN` : GitHub oauth token used for posting comments and settings the label.

## Extracting arguments
A regex pattern is provided as an argument which is then used to parse the comment into separate arguments. Each argument is written to a file. Additionally, some internal args are automatically set, eg. `PR_NUMBER`.

Using [regex named groups](https://godoc.org/regexp/syntax) is mandatory so that each env file is named after the regex group.

For example, the following regex will create a file named `RELEASE` with the content of the capture group:
```
(?mi)^/prombench\s*(?P<RELEASE>master|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$
```

The comment parsing is optional and is disabled when no regex is provided.

## Running as a webhook
Running comment monitor with the `--webhook` flag starts it in the webhook mode, it also requires the eventmap file which is specified by the `--eventmap` flag.

Example content of the `eventmap.yml` file:
```
- event_type: prombench_stop
  regex_string: (?mi)^/prombench\s+cancel\s*$
  comment_template: |
    Benchmark cancel is in progress.
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
