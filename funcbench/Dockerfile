FROM golang:alpine AS builder

RUN apk add git

ENV GO111MODULE=on

WORKDIR /
COPY . /

RUN CGO_ENABLED=0 go build main.go

FROM golang:alpine

RUN apk add git

COPY --from=builder /main /main



ENTRYPOINT ["/main"]

