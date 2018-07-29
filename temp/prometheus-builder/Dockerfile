FROM golang:1.10.2
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN \
    apt-get update && apt-get install -y --no-install-recommends \
        apt-utils \
        build-essential \
        ca-certificates \
        make \
        git \
	&& rm -rf /var/lib/apt/lists/*

RUN mkdir -p /go/src/github.com
COPY build.sh /go/src/github.com/build.sh
RUN chmod +x /go/src/github.com/build.sh

ENTRYPOINT /go/src/github.com/build.sh