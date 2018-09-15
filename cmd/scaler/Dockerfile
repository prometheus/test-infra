FROM alpine:3.8

COPY scaler scaler
COPY tmp/2_fake-webserver.yaml 2_fake-webserver.yaml

ENTRYPOINT ["/scaler", "scale","-f", "/2_fake-webserver.yaml"]