FROM golang AS builder

RUN apt-get install git -y

ENV GO111MODULE=on

WORKDIR /
COPY . /

RUN CGO_ENABLED=0 go build main.go

FROM golang

RUN apt-get install git -y

COPY --from=builder /main /main



ENTRYPOINT ["/main"]

