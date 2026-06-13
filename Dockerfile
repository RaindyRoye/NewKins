# Stage 1: Build
FROM golang:1.25-alpine AS builder

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
FROM alpine:3.21

WORKDIR /app

# Install runtime dependencies (git is needed for CI/CD operations)
RUN apk add --no-cache ca-certificates git tzdata

# Create non-root user for security
RUN addgroup -S gokins && adduser -S gokins -G gokins

# Copy binary from builder
COPY --from=builder /gokins /app/gokins

# Create data directory with proper ownership
RUN mkdir -p /data && chown gokins:gokins /data

# Switch to non-root user
USER gokins

# Expose ports (web + hbtp)
EXPOSE 8030 9711

# Health check using the /healthz endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8030/healthz || exit 1

# Labels
LABEL org.opencontainers.image.title="NewKins" \
      org.opencontainers.image.description="NewKins CI/CD Platform" \
      org.opencontainers.image.version="${VERSION}"

# Run
ENTRYPOINT ["/app/gokins", "run", "--web", ":8030"]
