# NOTE: building this requires it to have the build context of the repository root
FROM golang:1.12-alpine AS builder

RUN apk update && apk add --no-cache git \
    ca-certificates

WORKDIR /app
COPY tools/commentMonitor/*.go ./
COPY go.* ./

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -o /go/bin/commentMonitor

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/commentMonitor /commentMonitor
ENTRYPOINT ["/commentMonitor"]
