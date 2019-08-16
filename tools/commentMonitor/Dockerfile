# NOTE: building this requires it to have the build context of the repository root
FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git \
    ca-certificates

ENV GO111MODULE=on

WORKDIR /app
COPY tools/commentMonitor/main.go .

COPY go.mod .
COPY go.sum .
RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/commentMonitor

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/commentMonitor /commentMonitor
ENTRYPOINT ["/commentMonitor"]
