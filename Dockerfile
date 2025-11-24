# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go module files and download deps (none extra, but good pattern)
COPY go.mod ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary
RUN go build -o server main.go

# Run stage
FROM alpine:3.20

WORKDIR /app

# Copy the binary
COPY --from=builder /app/server .

# Copy static files
COPY static ./static

# Expose port (Render sets PORT env var)
EXPOSE 3000

# Start the server
CMD ["./server"]
