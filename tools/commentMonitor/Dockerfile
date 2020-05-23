FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY ./commentMonitor /bin/commentMonitor

ENTRYPOINT ["/bin/commentMonitor"]
