# Gokins: Lightweight CI/CD Pipeline Tool

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go)
![License](https://img.shields.io/badge/License-MIT-blue.svg)
![Build](https://img.shields.io/badge/build-passing-brightgreen)

Gokins is a lightweight, self-hosted continuous integration and continuous delivery (CI/CD) tool built with Go and Vue. It supports pipeline-as-code via YAML configuration, integrates with major Git platforms (GitHub, GitLab, Gitee, Gitea), and provides artifact management.

## Features

- **Continuous Integration & Delivery** - Pipeline-as-code with YAML configuration, stages, and steps
- **Multi-Platform Support** - Works on Linux, macOS, Windows
- **Git Platform Integration** - GitHub, GitLab, Gitee, Gitea webhook support
- **Artifact Management** - Store and manage build artifacts
- **Self-Hosted** - No data collection, runs entirely on your infrastructure
- **Database Support** - SQLite (default), MySQL, PostgreSQL

## Quick Start

### Prerequisites

- Go 1.22+ (for building from source)
- SQLite (built-in) or MySQL / PostgreSQL

### Build from Source

```bash
# Clone the repository
git clone https://github.com/RaindyRoye/NewKins.git gokins
cd gokins

# Build
make build

# Run
./gokins run
```

### Run with Docker

```bash
make docker
docker run -p 8030:8030 -v ~/.gokins:/root/.gokins gokins:latest
```

### Binary Download

Pre-built binaries are available at the [Releases](https://github.com/RaindyRoye/NewKins/releases) page.

### Installation

After starting Gokins, visit `http://localhost:8030` to complete the installation wizard.

Default admin credentials:
- Username: `gokins`
- Password: `123456`

## Configuration

Gokins reads configuration from `~/.gokins/app.yml`:

```yaml
datasource:
  driver: sqlite3
  url: ~/.gokins/gokins.db
server:
  runLimit: 5  # max concurrent builds
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GOKINS_WORKPATH` | Working directory for Gokins data | `~/.gokins` |
| `GOKINS_NOTUPDATEPASS` | Prevent password updates | `false` |

## Pipeline Configuration

Define pipelines using YAML:

```yaml
version: 1.0
vars:
stages:
  - stage:
    displayName: build
    name: build
    steps:
      - step: shell@sh
        displayName: test-build
        name: build
        commands:
          - echo Hello World
```

See the [YML Documentation](http://gokins.cn/工作流语法/) for full syntax reference.

## API Documentation

Gokins exposes a RESTful API documented with OpenAPI 3.0. The full specification is available at [`docs/openapi.yaml`](docs/openapi.yaml).

Key endpoints:
| Endpoint | Method | Description |
|----------|--------|-------------|
| `/healthz` | GET | Liveness probe |
| `/readyz` | GET | Readiness probe (DB + cache) |
| `/api/lg/login` | POST | Authenticate and get JWT token |
| `/api/pipeline/run` | POST | Trigger a pipeline build |
| `/api/runtime/build` | POST | Live build status |
| `/trigger/hook/:id` | POST | Git webhook receiver |

## Health Checks

Gokins provides health check endpoints for monitoring:

- `GET /healthz` - Basic liveness check, returns version info
- `GET /readyz` - Readiness check, verifies database and cache connectivity

## CLI Commands

```
gokins [command]

Available Commands:
  completion  Generate shell autocompletion
  daemon      Run as background daemon
  help        Help about any command
  run         Run in foreground
  version     Show version info

Flags:
  -h, --help             Help for gokins
      --nupass           Prevent password updates
      --web string       Web host (default ":8030")
  -w, --workdir string   Working directory
```

## Project Structure

```
├── cmd/          - CLI entrypoint (Cobra commands)
├── comm/         - Shared configuration and database access
├── engine/       - Build engine and task scheduling
├── model/        - Database models (XORM)
├── route/        - HTTP route handlers (Gin controllers)
├── server/       - Server initialization and setup
├── service/      - Business logic layer
├── hook/         - Webhook handlers (GitHub, GitLab, etc.)
├── thirdapi/     - Third-party API clients
├── util/         - Utility functions
├── bean/         - Request/response DTOs
├── migrates/     - Database migrations
├── docs/         - API documentation (OpenAPI spec)
├── Makefile      - Build automation
└── Dockerfile    - Multi-stage Docker build
```

## Development

```bash
# Run all tests
make test

# Run linter
make lint

# Format code
make fmt

# Clean build artifacts
make clean
```

## Architecture

Gokins uses:
- **Gin** for HTTP routing
- **XORM** for ORM/database operations
- **bbolt** for embedded key-value cache
- **Cobra** for CLI commands
- **golang-migrate** for database migrations

## Links

- **Website**: http://gokins.cn
- **Demo**: http://demo.gokins.cn (guest / 123456)

## License

MIT License - see [LICENSE](LICENSE) for details.
