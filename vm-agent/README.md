# VM Agent

A fake agent that generates realistic v4.IndexReport data and sends it over vsock to simulate a virtual machine compliance agent.

## Overview

This agent generates fake index reports containing:
- Package information (openssl, curl, systemd, bash, etc.)
- Distribution information (Ubuntu, RHEL variants)
- Repository information (Ubuntu Main, RHEL BaseOS, etc.)

The agent connects to the host via vsock and sends protobuf-serialized v4.IndexReport messages periodically.

## Building

### For amd64 (Linux):
```bash
make build-amd64
```

### For local architecture:
```bash
make build-local
```

### Clean build artifacts:
```bash
make clean
```

## Usage

```bash
# Run with default settings (port 1024, 10 packages)
./vm-agent-amd64

# Run with custom port
./vm-agent-amd64 -port 2048

# Run with custom number of packages
./vm-agent-amd64 -packages 5

# Run with both custom port and package count
./vm-agent-amd64 -port 2048 -packages 20
```

### CLI Options

- `-port`: vsock port to connect to (default: 1024)
- `-packages`: number of packages to include in fake reports (default: 10, max: 30)

## Features

- Generates realistic package data with variations
- Configurable number of packages per report (1-30 packages)
- Supports multiple Linux distributions (Ubuntu 20.04/22.04, RHEL 8/9)
- Randomly selects different repository configurations
- Sends reports every 10 seconds
- Graceful shutdown on SIGINT/SIGTERM
- Protobuf serialization for efficient transmission
- Comprehensive logging with emojis for easy monitoring

## Architecture

The agent is designed to run inside a virtual machine and communicate with the host via vsock. The vsock CID is not included in the generated reports as it will be added by the relay service on the host side.

## Dependencies

- `github.com/mdlayher/vsock` - vsock communication
- `google.golang.org/protobuf` - protobuf serialization
- `github.com/stackrox/rox` - StackRox protobuf definitions
