FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git 
# WORKDIR $GOPATH/src/mypackage/myapp/
COPY main.go .

RUN go get -d -v

RUN CGO_ENABLED=0 GOOS=linux go build -o /go/bin/sargs

FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/sargs /sargs
ENTRYPOINT ["/sargs"]
