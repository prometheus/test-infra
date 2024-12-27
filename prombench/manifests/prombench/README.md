## Prombench Benchmark Scenario Configuration

> NOTE(bwplotka): This is a custom scenario that changes the scrape preference to PrometheusProto ONLY for PR Prometheus.

This directory contains resources that are applied (and cleaned) on every benchmark request
via `infra` CLI using [`make deploy`](../../Makefile) and cleaned using [`make clean`](../../Makefile).

It assumes running cluster was created via `infra` CLI using `make cluster_create` and `make cluster_delete`.

### Customizations

#### Benchmarking from the custom test-infra commit/branch 

> NOTE: See https://github.com/prometheus/proposals/pull/41 for design.

On the `master` branch, in this directory, we maintain the standard, single benchmarking scenario used
as an acceptance validation for Prometheus. It's important to ensure it represents common Prometheus configuration.

The only user related parameter for the standard scenario is `RELEASE` version.

However, it's possible to create, a fully custom benchmarking scenarios for `/prombench` via `--bench.version=<branch|@commit>` flag.

Here are an example steps:

1. Create a new branch on https://github.com/prometheus/test-infra e.g. `benchmark/scenario1`.
2. Modify this directory to your liking e.g. changing query load, metric load of advanced Prometheus configuration. It's also possible to make Prometheus deployments and versions exactly the same, but vary in a single configuration flag, for feature benchmarking.

   > WARN: When customizing this directory, don't change `1a_namespace.yaml` or `1c_cluster-role-binding.yaml` filenames as they are used for cleanup routine. Or, if you change it, know what you're doing in relation to [`make clean` job](../../Makefile).

3. Push changes to the new branch.
4. From the Prometheus PR comment, call prombench as `/prombench <release> --bench.version=benchmark/scenario1` or `/prombench <release> --bench.version=@<relevant commit SHA from the benchmark/scenario1>` to use configuration files from this custom branch.

Other details:

* Other custom branch modifications other than to this directory do not affect prombench (e.g. to infra CLI or makefiles).
* `--bench.version` is designed for a short-term or even one-off benchmark scenario configurations. It's not designed for long-term, well maintained scenarios. For the latter reason we can later e.g. maintain multiple `manifests/prombench` directories and use it via [`--bench.directory` flag](#benchmarking-from-the-different-directory).
* Non-maintainers can follow similar process, but they will need to ask maintainer for a new branch and PR review. We can consider extending `--bench.version` to support remote repositories if this becomes a problem.
* Custom benchmarking logic is implemented in the [`maybe_pull_custom_version` make job](../../Makefile) and invoked by the prombench GH job on Prometheus repo on `deploy` and `clean`.

#### Benchmarking from the different directory.

On top of the commit/branch you can also specify custom directory with `--bench.directory` (default to this directory, so `manifests/prombench` value). This is designed if we even want to maintain standard benchmark modes for longer time e.g. agent mode.

For one-off benchmarks prefer one-off branches.

### Variables

It expects the following templated variables:

* `.PR_NUMBER`: The PR number from which `/prombench` was triggered. This PR number also tells what commit to use for the `prometheus-test-pr-{{ .PR_NUMBER }}` Prometheus image building (in the init container).
* `.RELEASE`: The argument provided by `/prombench` caller representing the Prometheus version (docker image tag for `quay.io/prometheus/prometheus:{{ .RELEASE }}`) to compare with, deployed as the `prometheus-test-{{ .RELEASE }}`.
* `.DOMAIN_NAME`
* `.LOADGEN_SCALE_UP_REPLICAS`
* `.GITHUB_ORG`
* `.GITHUB_REPO`
