FROM debian:sid
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN \
    apt-get update && apt-get install -y --no-install-recommends \
        apt-utils \
        build-essential \
        ca-certificates \
        make \
	&& rm -rf /var/lib/apt/lists/*

COPY prombench /bin/prombench
RUN mkdir -p /prombench/components/prombench/manifests

COPY Makefile /prombench/Makefile
COPY components/prombench/nodepools.yaml /prombench/components/prombench/nodepools.yaml
COPY components/prombench/manifests/benchmark /prombench/components/prombench/manifests/benchmark

WORKDIR /prombench
