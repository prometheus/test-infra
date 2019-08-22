FROM        quay.io/prometheus/busybox:latest
MAINTAINER  The Prometheus Authors <prometheus-developers@googlegroups.com>

ARG ARCH="amd64"
ARG OS="linux"

COPY .build/${OS}-${ARCH}/fake-webserver       /bin/webserver

EXPOSE     8080
ENTRYPOINT [ "/bin/webserver" ]
