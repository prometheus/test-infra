# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Prometheus test infrastructure: automated E2E testing and benchmarking (prombench) for Prometheus on Kubernetes (GKE, EKS, KIND). Includes CLI tools for cluster management, GitHub webhook handling, load generation, and alert notification.

## Build & Development Commands

```bash
# Build all binaries (requires promu)
make build

# Run all checks (style, license, lint, build, test, docs-check)
make all

# Run tests (uses -race on amd64)
make test

# Run a single test
go test -run TestName ./pkg/provider/...

# Lint
make lint                  # golangci-lint v2
make format                # goimports + gofumpt

# Docker
make docker                # build image
make docker-publish        # push image

# Prombench config validation
make -C prombench check_config
```

Build tool `promu` is auto-installed to `$GOPATH/bin`. Configuration in `.promu.yml` defines 7 binaries: `infra`, `tools/amGithubNotifier`, `tools/comment-monitor`, `tools/fake-webserver`, `tools/scaler`, `tools/load-generator`, `tools/block-sync`.

## Code Style

- Formatting: gofumpt + goimports (local prefix: `github.com/prometheus/test-infra`)
- Linter config: `.golangci.yml` (golangci-lint v2)
- `github.com/pkg/errors` is blocked; use stdlib `errors`/`fmt` instead
- All `.go` files must have a copyright header (first 3 lines)
- Copyright headers must use `Copyright The Prometheus Authors` (no year)
- CLI parsing uses `kingpin.v2`

## Architecture

**`pkg/provider/`** — Cloud provider abstraction layer:
- `provider.go`: Core types (`DeploymentResource`, `Resource`), template parsing, retry logic. Manifest files support Go templates; files with `noparse` suffix skip templating.
- `gke/`, `eks/`, `kind/`: Provider implementations for Google, AWS, local Kubernetes
- `k8s/`: Shared Kubernetes client utilities (apply/delete manifests, readiness checks)

**`infra/`** — Main CLI entry point (`infra.go`). Uses kingpin to orchestrate cluster creation, deployment, and teardown via the provider abstraction.

**`tools/`** — Standalone services, each with its own Dockerfile:
- `comment-monitor`: GitHub webhook server listening for `/prombench` commands, dispatches `repository_dispatch` events
- `amGithubNotifier`: Routes Alertmanager alerts to GitHub issue comments
- `load-generator`: Generates query load against Prometheus instances, exports metrics
- `block-sync`: Syncs TSDB blocks to/from object storage
- `fake-webserver`: Mock HTTP server for testing
- `scaler`: Kubernetes deployment scaling utility

**`prombench/`** — Orchestration layer:
- `Makefile`: Drives the full benchmark lifecycle (cluster create → infra deploy → benchmark run → cleanup)
- `manifests/cluster-infra/`: Persistent monitoring stack (Prometheus, Grafana, Alertmanager, ingress)
- `manifests/prombench/`: Per-benchmark resources (node pools, benchmark workloads)
- Provider selected via `PROVIDER` env var (gke/eks/kind)

## CI

- **CircleCI**: Primary CI — runs `make all` + prombench config checks, publishes Docker images
- **GitHub Actions**: YAML linting, golangci-lint, Grafana dashboard generation
