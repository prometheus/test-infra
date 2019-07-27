# NOTE: building this requires it to have the build context of the repository root

FROM golang:1.12-alpine
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk add --update make

COPY prombench /prombench/prombench
COPY Makefile /prombench/Makefile
COPY Makefile.common /prombench/Makefile.common
COPY manifests/prombench /prombench/manifests/prombench

WORKDIR /prombench
