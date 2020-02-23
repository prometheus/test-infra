## Intro
A tool used as a github action to run a `go test -bench` and compare changes from a PR against another branch. The benchmark is triggered by creating a comment which specifies a branch to compare. The results are then posted back as a PR comment.
Comparison is done using [benchcmp](https://godoc.org/golang.org/x/tools/cmd/benchcmp). Arguments for the benchcmp are read from files created by previous action (for example [commentMonitor](/tools/commentMonitor/main.go)), which is responsible for the comment parsing.

## Use
Tests are triggered by posting a comment in a PR with the following format:
`/funcbench <branch> <golang test regex>  [-no-race]`
Specifying which tests to run are filtered by using the standard golang regex format.
By default all benchmarks run with `-race` flag enabled and it can be disabled by appending `-no-race` at the end of the comment.

### Example Github actions workflow file
> Note: No longer using `issue_comment`, to be replaced with commentMonitor usage.
```
on: issue_comment // Workflow is executed when a pull request comment is created.
name: Benchmark
jobs:
  commentMonitor:
    runs-on: ubuntu-latest
    steps:
    - name: commentMonitor
      uses: docker://prominfra/comment-monitor:latest
      env:
        COMMENT_TEMPLATE: 'The benchmark has started.' // Body of a comment that is created to announce start of a benchmark.
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }} // Github secret token
      with:
        args: '"^/funcbench ?(?P<BRANCH>[^ B\.]+)? ?(?P<REGEX>\.|Bench.*|[^ ]+)? ?(?P<RACE>-no-race)?.*$"'
    - name: benchmark
      uses: docker://prominfra/funcbench:latest
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }} // Github secret token
```

## Set up
This tools is meant to be used as a Github action. The action itself is, to a large degree, unusable alone, as you need to combine it with another Github action that will provide necessary files to it. At this time, the only action it is supposed to work with, is [comment-monitor](https://github.com/prometheus/test-infra/tree/master/tools/commentMonitor).
- Create Github actions workflow file that is executed when an issue comment is created, `on = "issue_comment"`.
- Add comment-monitor Github action as a first step.
- Specify this regex `^/funcbench ?(?P<BRANCH>[^ B\.]+)? ?(?P<REGEX>\.|Bench.*|[^ ]+)? ?(?P<RACE>-no-race)?.*$` in the `args` field of the comment-monitor.
- Specify this Github action as a pre-built image, build from this source code, or just refer to this repository from the workflow file.
- Provide a Github token as an environment variable to both comment-monitor and funcbench.

## How to build
From the repository root:
`docker build -t <tag of your choice> -f funcbench/Dockerfile .`

## Examples
Execute benchmark named `FuncName` regex, and compare it with `master` branch.
 ```
 /funcbench master BenchmarkFuncName
 ```

Execute all benchmarks matching `FuncName.*` regex, and compare it with `master` branch.
 ```
 /funcbench master FuncName.*
 ```

Execute all benchmarks, and compere the results with `devel` branch.
 ```
 /funcbench devel .
 ```
