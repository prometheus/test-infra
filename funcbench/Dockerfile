FROM golang:1.13-alpine AS builder

WORKDIR /src

COPY funcbench/*.go ./
COPY go.* ./

RUN CGO_ENABLED=0 GO111MODULE=on go build -a -tags netgo -o /go/bin/funcbench

FROM golang:1.13-alpine
LABEL maintainer="The Prometheus Authors <prometheus-developers@googlegroups.com>"

RUN apk add git
COPY --from=builder /go/bin/funcbench /bin/

ENTRYPOINT ["/bin/funcbench"]
