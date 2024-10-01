FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY blockSyncer /bin/blocksyncer

ENTRYPOINT ["/bin/blocksyncer"]
