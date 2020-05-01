# funcbench

Benchmark and compare your Go code between commits or sub benchmarks. It automates the use of `go test -bench` to run the benchmarks and uses [benchcmp](https://godoc.org/golang.org/x/tools/cmd/benchcmp) to compare them.

funcbench currently supports two modes, Local and GitHub. Running it in the Github mode also allows it to accept *a pull request number* and *a branch/commit* to compare against, which makes it suitable for automated tests.

## Environment variables

- `GITHUB_TOKEN`: Access token to post benchmarks results to respective PR.

## Usage Examples

> Clean git state is required.

|Usage|Command|
|--|--|
|Execute benchmark named `BenchmarkFuncName` regex, and compare it with `master` branch. | ``` ./funcbench -v master BenchmarkFuncName ``` |
|Execute all benchmarks matching `BenchmarkFuncName.*` regex, and compare it with `master` branch.|```./funcbench -v master BenchmarkFuncName.*```|
|Execute all benchmarks, and compare the results with `devel` branch.|```./funcbench -v devel . ```|
|Execute all benchmarks matching `BenchmarkFuncName.*` regex, and compare it with `6d280faa16bfca1f26fa426d863afbb564c063d1` commit.|```./funcbench -v 6d280faa16bfca1f26fa426d863afbb564c063d1 BenchmarkFuncName.*```|
|Execute all benchmarks matching `BenchmarkFuncName.*` regex on current code. Compare it between sub-benchmarks (`b.Run`) of same benchmark for current commit. Errors out if there are no sub-benchmarks.|```./funcbench -v . FuncName.*```|
|Execute benchmark named `BenchmarkFuncName`, and compare `pr#35` with `master` branch.|```./funcbench --nocomment --github-pr="35" master BenchmarkFuncName```|

## Triggering with GitHub comments

The benchmark can be triggered by creating a comment in a PR which specifies a branch to compare. The results are then posted back to the PR as a comment.

The syntax is: `/funcbench <branch> <benchmark function regex>`

Examples:

* `/funcbench master BenchmarkQuery.*` - compare all the benchmarks mathching `BenchmarkQuery.*` for branch master vs the PR.

* `/funcbench feature-branch` or `/funcbench feature-branch .*` - compare all the benchmarks on feature-branch vs the PR.

* You can even add some comments along with the command.
    ```
    /funcbench old_branch .*

    The old_branch performs poorly, I bet mine are much better.
    ```

#### Setup

The comment is handled by [comment-monitor](https://github.com/prometheus/test-infra/tree/master/tools/commentMonitor) and then the parsed arguments are handed over to funcbench(if using Github Actions) or to [prombench](https://github.com/prometheus/test-infra/tree/master/prombench) if using funcbench with GKE.

- Create GitHub actions workflow file (see below) that is executed when an `repository_dispatch` event is on.
- Read BRANCH / BENCH_FUNC_REGEX / PR_NUMBER from event payload into environment variables.

#### Example GitHub action workflow file

```yaml
on: repository_dispatch
name: Funcbench Workflow
jobs:
  run_funcbench:
    name: Running funcbench
    if: github.event.action == 'funcbench_start'
    runs-on: ubuntu-latest
    env:
      AUTH_FILE: ${{ secrets.PROMBENCH_GKE_AUTH }}
      CLUSTER_NAME: << cluster name >>
      GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
      PROJECT_ID: << project id >>
      PR_NUMBER: ${{ github.event.client_payload.PR_NUMBER }}
      ZONE: << zone >>
    steps:
    - name: Prepare nodepool
      uses: docker://prominfra/prombench:latest
      with:
        args: make funcbench_nodepool_create
    - name: Run funcbench
      uses: docker://prominfra/prombench:latest
      env:
        BRANCH: ${{ github.event.client_payload.BRANCH }}
        GITHUB_ORG: prominfra
        GITHUB_REPO: prometheus
        GITHUB_TOKEN: ${{ secrets.PERSONAL_TOKEN }} # The GH action token lasts up to 60min so using PERSONAL_TOKEN guarantees that can post back the results even when the bench tests takes longer.
        BENCH_FUNC_REGEX: ${{ github.event.client_payload.BENCH_FUNC_REGEX }}
      with:
        args: make funcbench_resource_apply
    - name: Recycle all
      uses: docker://prominfra/prombench:latest
      with:
        args: make funcbench_resource_delete; make funcbench_nodepool_delete
```

## Building Docker container.

From the repository root:

`docker build -t <tag of your choice> -f funcbench/Dockerfile .`
