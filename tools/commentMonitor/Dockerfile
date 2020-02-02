FROM golang:1.13-alpine AS builder

WORKDIR /src
COPY tools/commentMonitor/*.go ./
COPY go.* ./

RUN GO111MODULE=on go build -a -tags netgo -o /go/bin/commentMonitor

FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY --from=builder /go/bin/commentMonitor /bin/

ENTRYPOINT ["/bin/commentMonitor"]
