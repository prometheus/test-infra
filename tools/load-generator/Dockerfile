FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY load-generator /bin/load-generator

EXPOSE 8080

ENTRYPOINT ["/bin/load-generator"]