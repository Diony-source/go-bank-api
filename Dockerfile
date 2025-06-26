FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/main ./cmd

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/main .

COPY config.yml .

COPY db/migrations ./db/migrations

EXPOSE 8080

CMD ["/app/main"]