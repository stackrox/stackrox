# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Testing Commands
* When validating changes in Go code, run `make go-unit-tests` and `make golangci-lint`.
* Run more targeted `go test ./...` commands for much faster feedback.
* Only run `make golangci-lint` when about to open a PR
- `make style` - Apply and check style standards (Go and JavaScript)
- `make ui-lint` - Lint UI code

### Individual Binary Builds
These targets build specific components as local binaries in the `bin/` directory, useful for faster development iteration:

- `make bin/central` - Build Central binary only
- `make bin/kubernetes` - Build Sensor binary only  
- `make bin/admission-control` - Build Admission Controller binary only
- `make bin/roxctl` - Build roxctl CLI binary only
- `make bin/migrator` - Build Migrator binary only
- `make bin/scanner` - Build Scanner binary only
- `make bin/scanner-v4` - Build Scanner v4 binary only
- `make bin/compliance` - Build Compliance binary only
- `make bin/upgrader` - Build Upgrader binary only
- `make bin/config-controller` - Build Config Controller binary only
- `make bin/installer` - Build Installer binary only
- `make bin/collector` - Build Collector binary (requires cmake)

### UI Development
- `make ui/build` - Build UI assets only

### Hotload Development
- `./hotload.sh sensor [namespace]` - Build and hotload Sensor binary into running pod
- `./hotload.sh central [namespace]` - Build and hotload Central binary into running pod
- If namespace is omitted, defaults to "default" namespace

### API Tests
To run E2E API tests from `tests/` directory:
```bash
# Set environment variables
export ROX_USERNAME=admin
export ROX_ADMIN_PASSWORD=letmein
export API_ENDPOINT=central.$(yq .namespace installer.yaml).svc:8000

# Run specific test
cd tests && go test -run <TestName>
```

### UI Development
From `ui/` directory:
- `npm ci` - Install UI dependencies
- `npm run deploy-local` - Deploy local StackRox instance for UI development

### Deployment Commands
- `bin/installer apply central` - Deploy central
- `bin/installer apply crs` - Apply CRS. Must wait for Central to be ready because it makes a call to Central API
- `bin/installer apply securedcluster` - Deploy Sensor, Collector, and Admission Control

## Architecture Overview

### Core Components
- **Central** (`central/main.go`) - Management control plane, web UI, API server, policy engine
- **Sensor** (`sensor/kubernetes/main.go`) - Kubernetes cluster agent, event monitoring, admission control
- **Admission Controller** (`sensor/admission-control/main.go`) - Webhook for deployment-time policy enforcement
- **Scanner** - Container vulnerability scanner (ClairCore-based)
- **Migrator** (`migrator/main.go`) - Database schema and data migration utility
- **roxctl** (`roxctl/main.go`) - Command-line administrative tool

### Communication Patterns
- **Central ↔ Sensor**: gRPC with mTLS, bidirectional streaming
- **Central ↔ Scanner**: gRPC/HTTP APIs for vulnerability data
- **UI ↔ Central**: REST/GraphQL APIs with token auth
- **Sensor ↔ Admission Controller**: gRPC within cluster
- **Admission Controller ↔ Central**: gRPC within mTLS, bidirectional streaming

### Key Technologies
- **Database**: PostgreSQL (primary datastore)
- **Communication**: gRPC with Protocol Buffers
- **Security**: mTLS everywhere, certificate rotation
- **Container Runtime**: Supports multiple Kubernetes distributions
- **UI**: React-based web interface

## Code Organization

### Go Modules Structure
- `central/` - Central service implementation with sub-modules for each domain
- `sensor/` - Sensor agent and admission controller
- `pkg/` - Shared libraries and utilities
- `generated/` - Auto-generated protobuf code and storage interfaces
- `migrator/` - Database migration tools
- `roxctl/` - CLI tool implementation
* `proto/` - Protocol Buffer files. API is generated from these files

### Build System
- Main `Makefile` with modular includes from `make/`
- Multi-architecture support (AMD64, ARM64, S390X)

### Testing Structure
- `tests/` - E2E API tests requiring deployed StackRox
- Unit tests co-located with source code
- Integration tests marked with build tags (e.g., `//go:build sql_integration`)
- QA backend tests in `qa-tests-backend/`

## Development Notes

### SQL Integration Tests
Require PostgreSQL server on port 5432:
```bash
docker run --rm --env POSTGRES_USER="$USER" --env POSTGRES_HOST_AUTH_METHOD=trust --publish 5432:5432 docker.io/library/postgres:13
```

### Local Development Flow

#### Traditional Flow
1. Ensure Kubernetes cluster is running (Docker Desktop, Colima, minikube)
2. Check context: `kubectl config current-context`
3. Build: `make image`
4. Deploy: `./deploy/deploy-local.sh`
5. Access UI: https://localhost:8000 (credentials in `deploy/k8s/central-deploy/password`)


### Important Files
- `BUILD_IMAGE_VERSION` - CI build image version
- `SCANNER_VERSION` - Scanner component version
- `COLLECTOR_VERSION` - Collector component version
- `go.mod` - Go dependencies (requires Go 1.23.4+)
- `installer.yaml` - Installer configuration (contains namespace setting)

## User-Specific Instructions

- @~/.claude/stackrox.md
