# NOTE: building this requires it to have the build context of the repository root
ARG ARCH="amd64"
ARG OS="linux"

FROM golang:1.12-alpine AS builder

WORKDIR /src
RUN apk add --update --no-cache git
COPY tools/amGithubNotifier ./
COPY go.* ./

RUN GO111MODULE=on go build -a -tags netgo -o /src/build/amGithubNotifier

FROM quay.io/prometheus/busybox-$OS-$ARCH:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"
COPY --from=builder /src/build/amGithubNotifier /bin/amGithubNotifier
ENTRYPOINT ["/bin/amGithubNotifier"]
