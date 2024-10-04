FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY block-sync /bin/blocksyncer

ENTRYPOINT ["/bin/blocksyncer"]
