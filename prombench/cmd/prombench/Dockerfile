# NOTE: building this requires it to have the build context of the repository root
FROM golang:1.12-alpine
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk add --update make git tini supervisor

ENV TEST_INFRA_DIR /test-infra
ENV TEST_INFRA_REPO https://github.com/prometheus/prombench.git
ENV PROMBENCH_DIR /prombench

COPY prombench /usr/bin/prombench
COPY cmd/prombench/supervisord.conf /etc/supervisord.conf
# supervisord programs
COPY cmd/prombench/supervisor-scripts/init.sh /usr/bin/init.sh
COPY cmd/prombench/supervisor-scripts/deploy.sh /usr/bin/deploy.sh
COPY cmd/prombench/supervisor-scripts/clean.sh /usr/bin/clean.sh
# supervisord eventlisteners
COPY cmd/prombench/supervisor-scripts/signal-supervisor.sh /usr/bin/signal-supervisor.sh

ENTRYPOINT ["/sbin/tini","--","supervisord", "-c", "/etc/supervisord.conf"]

WORKDIR $PROMBENCH_DIR