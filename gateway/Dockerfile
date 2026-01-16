# Build Stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway main.go

# Final Stage
FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/gateway .

# Expose port (Render sets PORT env var, typically 10000 or 8080)
EXPOSE 8080

CMD ["./gateway"]
