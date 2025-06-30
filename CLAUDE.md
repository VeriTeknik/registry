# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the MCP Registry - a community-driven registry service for Model Context Protocol (MCP) servers. It's a Go-based RESTful API that provides centralized discovery and management of MCP server implementations.

## Common Development Commands

### Building
```bash
# Docker build (recommended)
docker build -t registry .

# Direct Go build
go build ./cmd/registry

# Build publisher tool
cd tools/publisher && ./build.sh
```

### Running the Service
```bash
# Start with Docker Compose (includes MongoDB)
docker compose up

# Run directly (requires MongoDB running separately)
go run cmd/registry/main.go
```

### Testing
```bash
# Unit tests with coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./internal/...

# Integration tests
./integrationtests/run_tests.sh

# API endpoint tests (requires running server)
./scripts/test_endpoints.sh
```

### Linting
```bash
# Run golangci-lint
golangci-lint run --timeout=5m

# Check formatting
gofmt -s -l .
```

## Architecture Overview

### Core Components

1. **HTTP API Layer** (`internal/api/`)
   - Standard Go net/http server
   - RESTful endpoints: `/v0/health`, `/v0/servers`, `/v0/publish`, etc.
   - Swagger documentation generation
   - Request validation and error handling

2. **Service Layer** (`internal/service/`)
   - Business logic separation from HTTP handlers
   - Database abstraction through interfaces
   - Validation and data transformation

3. **Database Layer** (`internal/database/`)
   - Interface-based design supporting MongoDB and in-memory implementations
   - Repository pattern for data access
   - Automatic seed data import on startup

4. **Authentication** (`internal/auth/`)
   - GitHub OAuth integration for the publish endpoint
   - Bearer token validation
   - User verification against GitHub API

### Key Design Patterns

- **Dependency Injection**: Services receive dependencies through constructors
- **Interface-based Design**: Database and external services use interfaces for testability
- **Context Propagation**: All handlers and services accept context for cancellation/timeouts
- **Error Wrapping**: Consistent error handling with descriptive messages

### API Flow Example (Publish Endpoint)

1. Request hits `/v0/publish` handler in `internal/api/handlers.go`
2. Authentication middleware validates GitHub token
3. Handler parses and validates request body
4. Service layer (`internal/service/publish.go`) processes business logic
5. Database layer persists the server entry
6. Response formatted and returned to client

## Important Conventions

- **Go Module**: Uses Go 1.23 with module `github.com/modelcontextprotocol/registry`
- **Error Handling**: Always wrap errors with context using `fmt.Errorf`
- **Logging**: Use structured logging with appropriate levels
- **Testing**: Unit tests alongside code, integration tests in separate directory
- **API Versioning**: All endpoints prefixed with `/v0`
- **Database Collections**: MongoDB collections versioned (e.g., `servers_v2`)

## Environment Configuration

Key environment variables (prefix: `MCP_REGISTRY_`):
- `DATABASE_URL`: MongoDB connection string
- `SERVER_ADDRESS`: HTTP server bind address
- `GITHUB_CLIENT_ID/SECRET`: GitHub OAuth credentials
- `SEED_IMPORT`: Enable automatic seed data import
- `ENVIRONMENT`: Deployment environment (dev/test/prod)

## Development Tips

- Always run `go mod tidy` after adding dependencies
- Use `docker compose` for consistent development environment
- Check MongoDB indexes when adding new query patterns
- Update Swagger docs when modifying API endpoints
- Integration tests require a running MongoDB instance