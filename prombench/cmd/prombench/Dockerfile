# NOTE: building this requires it to have the build context of the repository root

FROM golang:1.13-alpine AS builder

WORKDIR /src

COPY pkg/ pkg/
COPY go.* ./
COPY prombench/cmd/prombench/*.go ./

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -tags netgo -o /go/bin/prombench

FROM alpine:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk add git

ENV TEST_INFRA_DIR /test-infra
ENV TEST_INFRA_REPO https://github.com/prometheus/test-infra.git

ENV PROMBENCH_DIR $TEST_INFRA_DIR/prombench

COPY --from=builder /go/bin/prombench /usr/bin/prombench
COPY prombench/cmd/prombench/clone.sh /usr/bin/clone.sh

ENTRYPOINT ["/usr/bin/clone.sh"]
