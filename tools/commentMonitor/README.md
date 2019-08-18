# commentMonitor - Inspect github comments to extract arguments from them

`commentMonitor` expects a github event payload as the input and based on the
regex provided it tries to extract arguments out of the comment. It can also can post
comments and set labels on the pr extracted from the github event payload.

See [`test_event.json`](./test_event.json) for an example of `issue_comment` payload.

#### Environment Variables:
- `COMMENT_TEMPLATE`: Accepts Golang template syntax
- `GITHUB_TOKEN` : GitHub oauth token
- `ANYTHING_ELSE`: Any other env var passed can be used in the template passed to `COMMENT_TEMPLATE`

## Extracting arguments
The content of the extracted arguments will be written to the filesystem in the `--output` directory. Use `--named-arg` flag to set the filename of these files.

If regex provided in `--named-arg` matches with some extracted argument, the filename will be set to that `--named-arg`.

example of setting `--named-arg`:
```
--named-arg=PR_NUMBER:"(?m)^([0-9]+)\s*$"
```

Disable it by setting `--no-args-extract` flag

## Commenting on PR
A golang template should be passed to the `COMMENT_TEMPLATE` env var.

The template variables can be accessed with `{{ index .VARIABLE_NAME }}`, the template variables have all the args extracted by commentMonitor aswell all the environtment variables.

Disable it by setting `--no-post-comment` flag

## Setting label on PR
Specify a custom label with `--label-name`, set to `prombench` by default.

Disable it by setting `--no-label-set` flag

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
