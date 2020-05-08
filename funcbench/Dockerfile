FROM golang:1.13-alpine AS builder

WORKDIR /src

COPY funcbench/*.go ./
COPY go.* ./

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -tags netgo -o /go/bin/funcbench

FROM prominfra/prombench:master
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

WORKDIR /funcbench

ENV PROMBENCH_CMD prombench

COPY --from=builder /go/bin/funcbench /bin/
COPY funcbench/ ./

# Need 'eval' to prevent bash keywords be run as commands
RUN echo -e '#!/bin/sh\ncd /funcbench\neval "$@"' >/usr/bin/docker_entrypoint
RUN chmod u+x /usr/bin/docker_entrypoint

ENTRYPOINT ["/bin/funcbench"]
