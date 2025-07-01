FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod ./

# Copy source code
COPY . .

# Download dependencies
RUN go mod tidy && go mod download

# Build the application
RUN go build -o analytics-api ./cmd/api

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/analytics-api .

# Expose port
EXPOSE 8081

# Run the binary
CMD ["./analytics-api"]