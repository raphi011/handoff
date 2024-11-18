FROM golang:1.23.3-alpine AS builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.* .

RUN go mod download && go mod verify

COPY . .

RUN go build ./cmd/example-server-bootstrap

FROM alpine:latest

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

COPY --from=builder /app/example-server-bootstrap .

# this should be the one set in the chart
EXPOSE 1337

ENTRYPOINT ["./example-server-bootstrap"]
