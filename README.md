# Repository Overview

This repository contains tools and configuration files for testing and benchmarking used in the Prometheus project.

## Tools and Projects

### [`/prombench`](prombench/)

**Prombench** is a project for automated end-to-end (E2E) testing and benchmarking for Prometheus.

- **Description**: For full details, see [prombench/README.md](prombench/README.md).

## Building Tools from Source

To build the tools from source, ensure you have a working Go environment with modules enabled. Follow these steps:

1. **Install `promu`**:
   ```bash
   go install github.com/prometheus/promu@latest
   ```

2. **Build the project**:
   ```bash
   promu build
   ```
