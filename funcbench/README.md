# funcbench

Benchmark and compare your Go code between commits or sub benchmarks. It automates the use of `go test -bench` to run the benchmarks and uses [benchcmp](https://godoc.org/golang.org/x/tools/cmd/benchcmp) to compare them.

funcbench currently supports two modes, Local and GitHub. Running it in the Github mode also allows it to accept _a pull request number_ and _a branch/commit_ to compare against, which makes it suitable for automated tests.

## Environment variables

- `GITHUB_TOKEN`: Access token to post benchmarks results to respective PR.

## Usage Examples

> Clean git state is required.

[embedmd]:# (funcbench-flags.txt)
```txt
usage: funcbench [<flags>] <target> [<bench-func-regex>] [<packagepath>]

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
  <target>              Can be one of '.', tag name, branch name or commit SHA
                        of the branch to compare against. If set to '.',
                        branch/commit is the same as the current one; funcbench
                        will run once and try to compare between 2
                        sub-benchmarks. Errors out if there are no
                        sub-benchmarks.
  [<bench-func-regex>]  Function regex to use for benchmark.Supports RE2 regexp
                        and is fully anchored, by default will run all
                        benchmarks.
  [<packagepath>]       Package to run benchmark against. Eg. ./tsdb, defaults
                        to ./...

```

### Building Docker Image
```
docker build -t prominfra/funcbench:master .
```

## Triggering with GitHub comments
<!-- If you change the heading, please change the anchor at 7a_commentmonitor_configmap_noparse.yaml aswell. -->

The benchmark can be triggered by creating a comment in a PR which specifies a branch to compare. The results are then posted back to the PR as a comment. The Github Actions workflow for funcbench [can be found here](https://github.com/prometheus/prometheus/blob/master/.github/workflows/funcbench.yml).

The syntax is: `/funcbench <branch> <benchmark function regex>`

Examples:

- `/funcbench master BenchmarkQuery.*` - compare all the benchmarks mathching `BenchmarkQuery.*` for branch master vs the PR.
- `/funcbench feature-branch` or `/funcbench feature-branch .*` - compare all the benchmarks on feature-branch vs the PR.
- `/funcbench tag-name BenchmarkQuery.* ./tsdb` - compare all the benchmarks mathching `BenchmarkQuery.*` for feature-branch vs the PR in package `./tsdb`.
- Multiline:
```
/funcbench old_branch .*

The old_branch performs poorly, I bet mine are much better.
```
