# commentMonitor
Simple webhook server to parse GitHub comments and take actions based on the comment.

Currently it only works with [`issue_comment` event](https://developer.github.com/v3/activity/events/types/#issuecommentevent) coming from PRs.

### Environment Variables:
- `LABEL_NAME`: If set, will add the label to the PR.
- `GITHUB_TOKEN` : GitHub oauth token used for posting comments and settings the label.
- Any other environment variable used in any of the comment templates in `eventmap.yml`.

### Setting up the webhook server
Running commentMonitor requires the eventmap file which is specified by the `--eventmap` flag.

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
- Set the webhook server URL as the webhook URL in the repository settings and set the content type to `application/json`.

## Extracting arguments
The `regex_string` provided in `eventmap.yml` is used to parse the comment into separate arguments. Additionally, some internal args are automatically set, eg. `PR_NUMBER` and `LAST_COMMMIT_SHA`.

If `regex_string` contains a capturing groups, using [named groups](https://godoc.org/regexp/syntax) is mandatory so that each comment argument is named after the regex group.

For example, the following regex will create an argument named `RELEASE` with the content of the capture group:
```
(?mi)^/prombench\s*(?P<RELEASE>master|v[0-9]+\.[0-9]+\.[0-9]+\S*)\s*$
```

### Docker image build
From the repository root:
```
$ make docker DOCKERFILE_PATH=tools/commentMonitor/Dockerfile DOCKER_IMAGE_NAME=comment-monitor DOCKER_IMAGE_TAG=0.0.2
```

#### Usage and examples:
```
./commentMonitor --help
```
