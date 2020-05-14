FROM golang:1.13-alpine AS builder

WORKDIR /src

COPY pkg/ pkg/
COPY go.* ./
COPY infra/*.go ./

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -tags netgo -o /go/bin/infra

FROM golang:1.13-alpine
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk add git make

COPY --from=builder /go/bin/infra /usr/bin/infra

ENTRYPOINT ["/usr/bin/infra"]
