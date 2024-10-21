FROM alpine:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk --no-cache add libc6-compat

COPY ./block-sync /bin/block-sync

RUN chmod +x /bin/block-sync

ENTRYPOINT ["/bin/block-sync"]
