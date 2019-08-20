# commentMonitor - Inspect github comments to extract arguments from them

`commentMonitor` expects a github event payload as the input and based on the
regex provided it tries to extract arguments out of the comment. It can also can post
comments and set labels on the pr extracted from the github event payload.

See [`test_event.json`](./test_event.json) for an example of `issue_comment` payload.

#### Environment Variables:
- `COMMENT_TEMPLATE`: If set, will attempt to post a comment. Accepts Golang template syntax.
- `LABEL_NAME`: If set, will attempt to create a label. Accepts the name of the label.
- `GITHUB_TOKEN` : GitHub oauth token
- `ANYTHING_ELSE`: Any other env var passed can be used in the template passed to `COMMENT_TEMPLATE`

## Comment validation
A regex needs to be provided for validate the regex with the comment in the event payload.

## Extracting arguments
The content of the extracted arguments will be written to the filesystem in the `--output` directory with filenames set to `ARG_0`, `ARG_1` and so forth.

Use [named & numbered capturing group](https://godoc.org/regexp/syntax) if you want to set custom filename of an argument.

For example, the following regex will create a file named `RELEASE` with the content of the capture group:
```
(?mi)^/prombench\s*(?P<RELEASE>master|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$
```

Don't provide a regex argument to commentMonitor if commentValidation and argument extraction is not desired.

## Commenting on PR
A golang template should be passed to the `COMMENT_TEMPLATE` env var.

The template variables can be accessed with `{{ index .VARIABLE_NAME }}`, the template variables have all the args extracted by commentMonitor aswell all the environment variables.

## Setting label on PR
Specify a custom label with `LABEL_NAME`.

## Usage

### Example for building the docker image
From the repository root:
```
$ make docker DOCKERFILE_PATH=tools/commentMonitor/Dockerfile DOCKER_IMAGE_NAME=comment-monitor DOCKER_IMAGE_TAG=0.0.1
```

#### Usage and examples:
```
./commentMonitor --help
```
