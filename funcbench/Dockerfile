FROM golang:1.13-alpine AS builder

WORKDIR /src

COPY funcbench/*.go ./
COPY go.* ./

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -tags netgo -o /go/bin/funcbench

FROM prombench:master
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

WORKDIR /funcbench

COPY --from=builder /go/bin/funcbench /bin/
COPY funcbench/ ./

ENTRYPOINT ["/bin/funcbench"]
