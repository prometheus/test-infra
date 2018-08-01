### Prometheus-Builder

This is used for building prometheus binaries from Pull Requests and running them on containers.  
Prombench uses this to build binaries for the Pull Request being benchmarked.

### How to run

#### Kubernetes Pod

A sample deployment config can be found [here](components/prombench/manifests/benchmark/3_prometheus-test.yaml#L176)

#### Docker Container

```
mkdir -p config
echo \
"global:
  scrape_interval:     15s
scrape_configs:
  - job_name: 'prometheus'
    scrape_interval: 5s
    static_configs:
      - targets: ['localhost:9090']" > config/prometheus.yaml

docker build -t prometheus-builder .
docker run --rm -p 9090:9090 -v /absolute-path/to/config:/etc/prometheus/config -v data:/data prometheus-builder <PR_NUMBER> 
```
