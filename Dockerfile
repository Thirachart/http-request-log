# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies for sqlite3 (needs gcc)
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Copy source code
COPY . .

# Tidy up modules and generate go.sum
RUN go mod tidy

# Build the application with CGO enabled for sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -o main main.go

# Final stage
FROM alpine:latest

# Install sqlite runtime dependencies
RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Expose port
EXPOSE 8000

# Run the binary
CMD ["./main"]
