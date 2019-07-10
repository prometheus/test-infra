ARG ARCH="amd64"
ARG OS="linux"
FROM quay.io/prometheus/busybox-${OS}-${ARCH}:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY amGithubNotifier /bin/amGithubNotifier

ENTRYPOINT ["/bin/amGithubNotifier"]
