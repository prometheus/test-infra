FROM debian:sid
MAINTAINER  The Prometheus Authors <prometheus-developers@googlegroups.com>

RUN \
    apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        ca-certificates \
        make \
	    python-dev \
        python-pip \
        python-setuptools \
	&& rm -rf /var/lib/apt/lists/*

RUN \
	python -m pip install --upgrade pip \
	&& pip install wheel setuptools pyyaml jinja2-cli[yaml]

COPY prombench /bin/prombench
RUN mkdir -p /prombench

COPY Makefile /prombench/Makefile
COPY config /prombench/config
COPY manifests /prombench/manifests

WORKDIR /prombench