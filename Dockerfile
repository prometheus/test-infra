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
RUN mkdir -p /prombench

COPY Makefile /prombench/Makefile
COPY config /prombench/config
COPY manifests /prombench/manifests

WORKDIR /prombench