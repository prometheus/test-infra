FROM alpine:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk --no-cache add libc6-compat

COPY ./block-sync /bin/blocksyncer

RUN chmod +x /bin/blocksyncer

ENTRYPOINT ["/bin/blocksyncer"]
