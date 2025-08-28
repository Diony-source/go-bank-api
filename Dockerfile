# Stage 1: Build the application
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
# Download dependencies
RUN go mod download

# Copy the source code
COPY . .
# Build the application binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/main ./cmd

# Stage 2: Create the final, lightweight image
FROM alpine:latest

WORKDIR /app

# Copy the built binary from the 'builder' stage
COPY --from=builder /app/main .

# Copy necessary configuration and assets
COPY config.yml .
COPY db/migrations ./db/migrations
COPY docs ./docs

# Expose the port the app runs on
EXPOSE 8080

# Command to run the application
CMD ["/app/main"]