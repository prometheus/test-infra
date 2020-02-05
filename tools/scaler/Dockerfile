FROM golang:1.13-alpine AS builder

WORKDIR /src
COPY pkg/ pkg/
COPY tools/scaler/*.go ./
COPY go.* ./

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -tags netgo -o /go/bin/scaler

FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY --from=builder /go/bin/scaler /bin/

ENTRYPOINT ["/bin/scaler"]
