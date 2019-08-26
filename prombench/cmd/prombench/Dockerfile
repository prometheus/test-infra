# NOTE: building this requires it to have the build context of the repository root
FROM golang:1.12-alpine
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk add --update --no-cache make git tini

ENV TEST_INFRA_DIR /test-infra
ENV TEST_INFRA_REPO https://github.com/prometheus/prombench.git

ENV PROMBENCH_DIR /prombench

COPY prombench /usr/bin/prombench
COPY cmd/prombench/clone.sh /usr/bin/clone.sh

ENTRYPOINT ["/sbin/tini", "--", "/usr/bin/clone.sh"]

WORKDIR $PROMBENCH_DIR