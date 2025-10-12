# StackRox VM Agent

A Go-based virtual machine agent that collects RPM package information and transmits it to StackRox for vulnerability analysis.

## Overview

The VM agent performs the following operations:
1. Collects RPM package information using `rpm -qa`
2. Parses OS information from `/etc/os-release`
3. Generates proper CPE identifiers for vulnerability matching
4. Builds a Scanner V4 IndexReport with package data
5. Transmits the report via VSOCK (default) or gRPC

## Usage

### VSOCK Mode (Default)
```bash
# Connect via VSOCK to host
./agent
```

### gRPC Mode
```bash
# Connect via gRPC with TLS certificates
./agent --mode=grpc --cert-path=/path/to/certs --sensor-url=https://sensor.stackrox.svc:443
```

## Requirements

### VSOCK Mode
- VM must have `autoattachVSOCK: true` configured
- `/dev/vsock` device must be available
- Host must be running VSOCK listener on port 1024

### gRPC Mode
- TLS certificates in specified cert path:
  - `cert.pem` - Client certificate
  - `key.pem` - Client private key
  - `ca.pem` - CA certificate (optional)
- Network connectivity to sensor URL

## Command Line Options

- `--mode`: Transmission mode (`vsock` or `grpc`, default: `vsock`)
- `--cert-path`: Path to TLS certificates directory (required for gRPC mode)
- `--sensor-url`: StackRox sensor URL (required for gRPC mode)

## Architecture

The agent is structured into several internal modules:

- **rpm**: RPM package collection and parsing
- **cpe**: CPE generation and OS information parsing
- **report**: IndexReport construction
- **vsock**: VSOCK communication
- **grpc**: gRPC communication with TLS

## Building

```bash
cd /root/workspace/src/stackrox
go build -o agent ./agent
```

## Integration

The agent integrates with StackRox through:
- **Protobuf APIs**: Uses `virtualmachine.v1.IndexReport` and `scanner.v4.*` types
- **VSOCK**: Connects to existing VSOCK listener infrastructure
- **gRPC**: Uses `VirtualMachineIndexReportService` for transmission

## Error Handling

The agent performs validation for:
- VSOCK device availability for VSOCK mode
- Required arguments for gRPC mode
- RPM command execution
- Network connectivity and TLS certificate validation

Errors are logged with descriptive messages to aid in troubleshooting.