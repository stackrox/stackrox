# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Information

**Upstream Repository**: https://github.com/stackrox/stackrox  
**Legacy Upstream Repository**: https://github.com/stackrox/rox - archived, but may contain valuable past information

## Common Development Commands

### Build Commands
- `make image` - Build the main StackRox container image with tag from `make tag`
- `make main-build-dockerized` - Compile all Go binaries using Docker
- `make main-build` - Build main binaries with dependency prep
- `make cli` - Build and install roxctl CLI for all platforms
- `make cli_host-arch` - Build roxctl CLI for current platform only
- `make all-builds` - Build all components (CLI, main, UI, docs)

### Testing Commands
- `make test` - Run all tests (Go unit tests, UI tests, shell tests)
- `make go-unit-tests` - Run Go unit tests only
- `make go-postgres-unit-tests` - Run PostgreSQL integration tests (requires running Postgres on port 5432)
- `make ui-test` - Run UI tests
- `make ui-component-tests` - Run UI component tests
- `make shell-unit-tests` - Run shell script tests

### Development Commands
- `make fast-central` - Quickly recompile and restart Central component
- `make fast-sensor` - Quickly recompile Sensor component
- `make fast-migrator` - Quickly recompile migrator component

### Code Quality Commands
- `make style` - Run all style checks (Go, protobuf, shell)
- `make golangci-lint` - Run Go linter
- `make proto-style` - Check protobuf style
- `make shell-style` - Check shell script style

### Code Generation Commands
- `make proto-generated-srcs` - Generate Go code from protobuf definitions
- `make go-generated-srcs` - Generate Go code (mockgen, stringer, easyjson)
- `make generated-srcs` - Generate all source code

### Local Development Commands
- `./deploy/deploy-local.sh` - Deploy StackRox locally (requires existing k8s cluster)
- `make install-dev-tools` - Install development tools (linters, generators)

### Single Test Examples
- Run specific Go test: `go test -v ./central/path/to/package -run TestSpecificFunction`
- Run PostgreSQL integration tests: `go test -v -tags sql_integration ./central/path/to/package`

## Architecture Overview

StackRox is a Kubernetes-native security platform with a distributed microservices architecture:

### Core Components
- **Central** (`/central/`) - Go-based API server, policy engine, and management hub with PostgreSQL storage
- **Sensor** (`/sensor/`) - Go-based Kubernetes monitoring agent deployed per cluster
- **Scanner** (`/scanner/`) - Go-based vulnerability scanning service using ClairCore
- **UI** (`/ui/`) - React/TypeScript frontend with modern web stack
- **roxctl** (`/roxctl/`) - Go-based CLI tool for administration and CI/CD integration
- **Operator** (`/operator/`) - Kubernetes operator for lifecycle management

### Technology Stack
- **Backend**: Go (1.24.0+), PostgreSQL, gRPC/HTTP APIs, Kubernetes controllers
- **Frontend**: React, TypeScript, Node.js (20.0.0+), npm
- **Infrastructure**: Kubernetes-native, Helm charts, Docker/Podman containers
- **Communication**: mTLS for security, gRPC for internal services, REST APIs for external access

### Deployment Model
- **Central Services**: Deployed in management cluster (Central, Scanner, UI, Database)
- **Secured Cluster Services**: Deployed per monitored cluster (Sensor, Admission Controller)
- **Multi-cluster support**: One Central instance monitors multiple Kubernetes clusters

### Key Directories
- `/central/` - Central management service code
- `/sensor/` - Sensor agent code for cluster monitoring
- `/scanner/` - Vulnerability scanning service
- `/ui/` - Web frontend application
- `/roxctl/` - Command-line interface
- `/operator/` - Kubernetes operator
- `/generated/` - Auto-generated code from protobuf definitions
- `/proto/` - Protocol buffer definitions
- `/pkg/` - Shared Go libraries and utilities
- `/deploy/` - Deployment scripts and configurations
- `/qa-tests-backend/` - Integration tests (Groovy/Spock)

### Development Workflow
1. Use `make install-dev-tools` to set up development environment
2. Run `make proto-generated-srcs` when protobuf files change
3. Use `make fast-central` or `make fast-sensor` for quick development iterations
4. Run `make style` before committing to ensure code quality
5. Use `./deploy/deploy-local.sh` for local testing with existing k8s cluster

### Environment Variables
- `STORAGE=pvc` - Persist PostgreSQL data between restarts
- `SKIP_UI_BUILD=1` - Skip UI builds to speed up development
- `SKIP_CLI_BUILD=1` - Skip CLI builds to speed up development
- `DEBUG_BUILD=yes` - Create debug build with debugging capabilities
- `MAIN_IMAGE_TAG` - Override default image tag for deployments

### Testing Notes
- PostgreSQL integration tests require Postgres running on port 5432
- Use `docker run --rm --env POSTGRES_USER="$USER" --env POSTGRES_HOST_AUTH_METHOD=trust --publish 5432:5432 docker.io/library/postgres:13` for test setup
- Integration tests in `/qa-tests-backend/` use Groovy/Spock framework
- Tests marked with `//go:build sql_integration` require database connectivity

### Style and Conventions
- Go code follows golangci-lint standards
- Additional project specific style guide in `.github/go-coding-style.md`
- Protocol buffers have enforced style guidelines
- Shell scripts are checked with shellcheck
- UI code uses TypeScript with React conventions
- All generated code should not be manually edited
