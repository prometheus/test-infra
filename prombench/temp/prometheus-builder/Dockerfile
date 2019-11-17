FROM alpine:3.10.3
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk update && apk add --no-cache \
    ca-certificates \
    git \
    jq \
    bash \
    curl

COPY build.sh /
RUN chmod +x /build.sh

ENTRYPOINT ["/build.sh"]
