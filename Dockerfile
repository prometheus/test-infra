FROM golang:1.10.2
MAINTAINER  The Prometheus Authors <prometheus-developers@googlegroups.com>

COPY benchmark /bin/benchmark
RUN mkdir -p /prometheus

COPY spec.example.yaml /prometheus/spec.example.yaml
COPY manifests /prometheus/manifests

WORKDIR    /prometheus
ENTRYPOINT ["/bin/benchmark"]