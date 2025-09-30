# VM Agent

Runs inside VMs to scan for vulnerabilities and report back to the host via vsock.

## What it does

Scans the VM for installed packages (RPM/DNF databases), creates vulnerability reports, and sends them to the host over vsock. Can run once or continuously in daemon mode.

## Usage

```bash
# Single scan
./agent

# Daemon mode (scans every 5 minutes)
./agent --daemon

# Custom settings
./agent --daemon --index-interval 10m --host-path /custom/path --port 2048
```

## Flags

- `--daemon` - Run continuously (default: false).
- `--index-interval` - Time between scans in daemon mode (default: 5m).
- `--host-path` - Where to look for package databases (default: /).
- `--port` - VSock port (default: 1024).

## How it works

1. Scans filesystem for RPM/DNF package databases.
2. Pulls repo-to-CPE mappings from Red Hat.
3. Creates protobuf index report.
4. Sends report to host via vsock.

The host receives these reports and forwards them to StackRox Central for vulnerability analysis.

## Building

```bash
go build -o agent .

# For Linux VMs
GOOS=linux GOARCH=amd64 go build -o agent-linux .
```

## Troubleshooting

**Can't connect to host**
- Check if vsock is enabled in the VM.
- Verify the port isn't in use.
- Make sure vsock kernel modules are loaded.

**No packages found**
- Check `--host-path` points to the right place.
- Verify RPM/DNF databases exist and are readable.

**Scan failures**
- Check internet access for repo-to-CPE downloads.
- Look at logs for specific errors.
