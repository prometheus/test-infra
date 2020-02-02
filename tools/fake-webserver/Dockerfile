FROM golang:1.13-alpine AS builder

WORKDIR /src
COPY tools/fake-webserver/*.go ./
COPY go.* ./

RUN GO111MODULE=on go build -a -tags netgo -o /go/bin/fake-webserver

FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY --from=builder /go/bin/fake-webserver /bin/webserver

EXPOSE 8080
ENTRYPOINT [ "/bin/webserver" ]
