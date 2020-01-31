FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"


COPY scaler /scaler

ENTRYPOINT ["/scaler"]
