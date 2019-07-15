# NOTE: building this requires it to have the build context of the repository root

FROM debian:sid
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN \
    apt-get update && apt-get install -y --no-install-recommends \
        apt-utils \
        build-essential \
        ca-certificates \
        make \
	&& rm -rf /var/lib/apt/lists/*

COPY prombench /prombench/prombench
COPY Makefile /prombench/Makefile
COPY Makefile.common /prombench/Makefile.common
COPY manifests/prombench /prombench/manifests/prombench

WORKDIR /prombench
ENTRYPOINT [ "make", "-C","/prombench/","deploy"]
