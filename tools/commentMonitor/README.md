# commentMonitor - Inspect github comments to extract arguments from them

`commentMonitor` expects a github event payload as the input and based on the
regex argument it extracts arguments out of the comment. It can can also post
comments and set labels on the pr from which the comment was received.

See [the github issue events api](https://developer.github.com/v3/issues/events/) for some examples.

#### Environment Variables:
- `COMMENT_TEMPLATE`: If set, will post a comment with the content. It uses the Golang template variables substitutions. If content text includes a variable name `{{ index .envVariable }}` that exists as an env variable it is expanded with the content of the variable.
- `LABEL_NAME`: If set, will add the label to the PR.
- `GITHUB_TOKEN` : GitHub oauth token used for posting comments and settings the label.

## Extracting arguments
A regex pattern is provided as an argument which is than used to parse the comment into separate arguments. Each argument is written to a file. 
Using [regex named groups](https://godoc.org/regexp/syntax) is mandatory so that each env file is named after the regex group.

For example, the following regex will create a file named `RELEASE` with the content of the capture group:
```
(?mi)^/prombench\s*(?P<RELEASE>master|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$
```

The comment parsing is optional and is disabled when no regex is provided.

### Docker image build
From the repository root:
```
$ make docker DOCKERFILE_PATH=tools/commentMonitor/Dockerfile DOCKER_IMAGE_NAME=comment-monitor DOCKER_IMAGE_TAG=0.0.1
```

#### Usage and examples:
```
./commentMonitor --help
```
