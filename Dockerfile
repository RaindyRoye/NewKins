# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git build-base ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build flags
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown
ARG VERSION=1.3.7

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-X 'github.com/gokins/gokins/comm.Version=${VERSION}' \
              -X 'github.com/gokins/gokins/comm.BuildTime=${BUILD_TIME}' \
              -X 'github.com/gokins/gokins/comm.GitCommit=${GIT_COMMIT}'" \
    -o /gokins ./cmd/...

# Stage 2: Run
FROM alpine:3.18

WORKDIR /app

# Install runtime dependencies (git is needed for CI/CD operations)
RUN apk add --no-cache ca-certificates git tzdata

# Copy binary from builder
COPY --from=builder /gokins /app/gokins

# Expose port
EXPOSE 8030

# Run
ENTRYPOINT ["/app/gokins", "run", "--web", ":8030"]
