# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code
COPY . .

# Build the extended application
RUN go build -o registry-extended ./extensions/cmd/registry/main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/registry-extended .
COPY --from=builder /app/data ./data

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./registry-extended"]