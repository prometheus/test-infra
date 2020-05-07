FROM golang:1.13-alpine AS builder

WORKDIR /src

COPY pkg/ pkg/
COPY go.* ./
COPY prombench/cmd/prombench/*.go ./

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -tags netgo -o /go/bin/prombench

FROM alpine:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

WORKDIR /prombench

RUN apk add git make

COPY --from=builder /go/bin/prombench /usr/bin/prombench
COPY prombench/ ./

# Need 'eval' to prevent bash keywords be run as commands
RUN echo -e '#!/bin/sh\ncd /prombench\neval "$@"' >/start.sh
RUN chmod u+x /start.sh

ENTRYPOINT ["/start.sh"]
