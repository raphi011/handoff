FROM golang:1.21-alpine

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.* .

RUN go mod download

COPY . .

RUN go build ./cmd/example

ENTRYPOINT ["./example"]
