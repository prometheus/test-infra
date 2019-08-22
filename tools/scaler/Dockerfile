FROM alpine:3.8

ARG ARCH="amd64"
ARG OS="linux"

COPY .build/${OS}-${ARCH}/scaler /scaler

ENTRYPOINT ["/scaler"]
