FROM quay.io/prometheus/golang-builder:1.24-base

RUN mkdir -p /go/src/github.com

COPY ./build.sh /go/src/github.com/build.sh

RUN chmod +x /go/src/github.com/build.sh

ENTRYPOINT ["/go/src/github.com/build.sh"]
