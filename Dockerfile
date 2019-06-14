FROM debian:sid
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN \
    apt-get update && apt-get install -y --no-install-recommends \
        apt-utils \
        build-essential \
        ca-certificates \
        make \
	&& rm -rf /var/lib/apt/lists/*

RUN mkdir -p /prombench/components/prombench/manifests

COPY prombench /prombench
COPY Makefile /Makefile
COPY manifests/prombench /manifests/prombench

WORKDIR /prombench
