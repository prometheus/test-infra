# funcbench

Benchmark and compare your Go code between commits or sub benchmarks. It automates the use of `go test -bench` to run the benchmarks and uses [benchcmp](https://godoc.org/golang.org/x/tools/cmd/benchcmp) to compare them.

funcbench currently supports two modes, Local and GitHub. Running it in the Github mode also allows it to accept *a pull request number* and *a branch/commit* to compare against, which makes it suitable for automated tests.

## Environment variables

- `GITHUB_TOKEN`: Access token to post benchmarks results to respective PR.

## Usage Examples

> Clean git state is required.

[embedmd]:# (funcbench-flags.txt)
```txt
usage: funcbench [<flags>] <target> [<bench-func-regex>]

Benchmark and compare your Go code between sub benchmarks or commits.

  * For BenchmarkFuncName, compare current with master: ./funcbench -v master BenchmarkFuncName
  * For BenchmarkFunc.*, compare current with master: ./funcbench -v master BenchmarkFunc.*
  * For all benchmarks, compare current with devel: ./funcbench -v devel .* or ./funcbench -v devel
  * For BenchmarkFunc.*, compare current with 6d280 commit: ./funcbench -v 6d280 BenchmarkFunc.*
  * For BenchmarkFunc.*, compare between sub-benchmarks of same benchmark on current commit: ./funcbench -v . BenchmarkFunc.*
  * For BenchmarkFuncName, compare pr#35 with master: ./funcbench --nocomment --github-pr="35" master BenchmarkFuncName
Flags:
  -h, --help                 Show context-sensitive help (also try --help-long
                             and --help-man).
  -v, --verbose              Verbose mode. Errors includes trace and commands
                             output are logged.
      --nocomment            Disable posting of comment using the GitHub API.
      --owner="prometheus"   A Github owner or organisation name.
      --repo="prometheus"    This is the repository name.
      --github-pr=GITHUB-PR  GitHub PR number to pull changes from and to post
                             benchmark results.
      --workspace="/tmp/funcbench"
                             Directory to clone GitHub PR.
      --result-cache="_dev/funcbench"
                             Directory to store benchmark results.
  -t, --bench-time=1s        Run enough iterations of each benchmark to take t,
                             specified as a time.Duration. The special syntax Nx
                             means to run the benchmark N times
  -d, --timeout=2h           Benchmark timeout specified in time.Duration
                             format, disabled if set to 0. If a test binary runs
                             longer than duration d, panic.

Args:
  <target>              Can be one of '.', branch name or commit SHA of the
                        branch to compare against. If set to '.', branch/commit
                        is the same as the current one; funcbench will run once
                        and try to compare between 2 sub-benchmarks. Errors out
                        if there are no sub-benchmarks.
  [<bench-func-regex>]  Function regex to use for benchmark.Supports RE2 regexp
                        and is fully anchored, by default will run all
                        benchmarks.

```

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
