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
COPY encode.txt /prombench/encode.txt
COPY encoded.txt /prombench/encoded.txt
# RUN echo $HOME
# RUN echo $CLUSTER_NAME
#RUN echo $AUTH_FILE | base64 -d > /prombench/auth.json

WORKDIR /prombench
ENTRYPOINT [ "make", "-C","/prombench/","deploy"]
#CMD echo $GITHUB_SHA
#CMD echo $AUTH_FILE | base64 -di > /prombench/auth.json
#CMD make deploy