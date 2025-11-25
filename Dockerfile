# ---------- Build stage ----------
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go module files and download deps
COPY go.mod ./
# If you have go.sum, copy it too:
# COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Go binary
# CGO_DISABLED=0 + GOOS=linux gives a static-ish binary that runs fine in alpine
RUN CGO_ENABLED=0 GOOS=linux go build -o server main.go

# ---------- Run stage ----------
FROM alpine:3.19

WORKDIR /app

# (Optional) install CA certs if your DB uses TLS
RUN apk add --no-cache ca-certificates

# Copy the binary from the builder
COPY --from=builder /app/server .

# Copy static files
COPY static ./static

# Default port the app listens on (for docs); Render sets PORT itself
EXPOSE 3000

# Default PORT for local run; Render will inject its own
ENV PORT=3000

# Start the server
CMD ["./server"]
