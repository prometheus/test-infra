FROM golang:1.13-alpine
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk add git make

COPY infra /bin/infra

ENTRYPOINT ["/bin/infra"]
