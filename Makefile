APP_NAME := gokins
GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION := 1.3.7

LDFLAGS := -ldflags="-X 'github.com/gokins/gokins/comm.Version=${VERSION}' -X 'github.com/gokins/gokins/comm.BuildTime=${BUILD_TIME}' -X 'github.com/gokins/gokins/comm.GitCommit=${GIT_COMMIT}'"

.PHONY: build clean test lint docker

# Build the binary
build:
	go build ${LDFLAGS} -o bin/${APP_NAME} ./cmd/...

# Run tests with coverage
test:
	go test -v -coverprofile=coverage.out ./...

# Run linter
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out

# Build docker image
docker:
	docker build -t ${APP_NAME}:${VERSION} .

# Format code
fmt:
	go fmt ./...

# Update dependencies
deps:
	go get -u ./...
	go mod tidy
