FROM quay.io/prometheus/golang-builder:1.23-main

RUN mkdir -p /go/src/github.com

COPY ./build.sh /go/src/github.com/build.sh

COPY ./key.sh /download-key/key.sh

RUN chmod +x /go/src/github.com/build.sh

ENTRYPOINT ["/go/src/github.com/build.sh"]
