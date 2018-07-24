FROM golang:1.10.2
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

# Installing Docker
RUN apt-get update
RUN apt-get install -y --no-install-recommends \
		apt-transport-https \
		build-essential \
		ca-certificates \
		curl \
		gnupg2 \
		libsnmp-dev \
		software-properties-common \
		git \
		make \
		&& rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
RUN add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable"

RUN apt-get update
RUN apt-get -y install docker-ce