FROM alpine:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk --no-cache add libc6-compat

COPY ./block-sync /bin/block-sync

# Copy the download.sh script into the container's /scripts directory
COPY ./download.sh /scripts/download.sh

# Ensure that the download.sh script is executable
RUN chmod +x /scripts/download.sh

RUN chmod +x /bin/block-sync

ENTRYPOINT [ "/scripts/download.sh" ]
