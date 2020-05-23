FROM quay.io/prometheus/busybox:latest
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

COPY ./fake-webserver /bin/webserver

EXPOSE 8080
ENTRYPOINT [ "/bin/webserver" ]
