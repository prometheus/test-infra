FROM quay.io/prometheus/busybox:latest

LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY ./comment-monitor /bin/comment-monitor

ENTRYPOINT ["/bin/comment-monitor"]
