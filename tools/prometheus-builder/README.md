### Prometheus-Builder

This is used for building prometheus binaries from Pull Requests and running them on containers.  
Prombench uses this to build binaries for the Pull Request being benchmarked.

### Building Docker Image
```
docker build -t prominfra/prometheus-builder:master .
```
