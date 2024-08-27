FROM golang:1.20-alpine

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o load-generator .

EXPOSE 8080

ENTRYPOINT ["./load-generator"]
