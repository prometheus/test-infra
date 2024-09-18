FROM quay.io/prometheus/busybox:latest

LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY ./amGithubNotifier /bin/amGithubNotifier

ENTRYPOINT ["/bin/amGithubNotifier"]
