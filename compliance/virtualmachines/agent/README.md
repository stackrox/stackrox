# VM Agent

Runs inside VMs to scan for vulnerabilities and report back to the host via vsock.
While not directly related to the `compliance` feature, the agent utilizes compliance
node scanning code for package scanning in the virtual machine.

## What it does

Scans the VM for installed packages (RPM/DNF databases), creates vulnerability reports, and sends them to the host over vsock. Can run once or continuously in daemon mode.

## Usage

```bash
# Single scan
sudo ./agent

# Daemon mode (scans every 5 minutes)
sudo ./agent --daemon

# Custom settings
sudo ./agent --daemon --index-interval 10m --host-path /custom/path --port 2048
```

## Flags

- `--daemon` - Run continuously (default: false).
- `--index-interval` - Time between scans in daemon mode (default: 5m).
- `--host-path` - Where to look for package databases (default: /).
- `--port` - VSock port (default: 1024).
- `--repo-cpe-url` - URL for the repository to CPE mapping.
- `--timeout` - VSock client timeout when sending index reports.
- `--verbose` - Prints the index reports to stdout.

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
