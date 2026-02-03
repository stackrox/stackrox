# VM Agent

Runs inside VMs to scan for vulnerabilities and report back to the host via vsock.
While not directly related to the `compliance` feature, the agent utilizes compliance
node scanning code for package scanning in the virtual machine.

## What it does

Scans the VM for installed packages (`rpm`/`dnf` databases), creates vulnerability reports, and sends them to the host over vsock. Can run once or continuously in daemon mode. Requires `sudo` privileges to scan packages.

## Usage

```bash
# Single scan
sudo ./roxagent

# Daemon mode (scans every 4 hours by default)
sudo ./roxagent --daemon

# Custom settings
sudo ./roxagent --daemon --index-interval 10m --host-path /custom/path --port 2048
```

## Flags

- `--daemon` - Run continuously (default: false).
- `--index-interval` - Time between scans in daemon mode (default: 4h).
- `--host-path` - Where to look for package databases (default: /).
- `--max-initial-report-delay` - Max delay before starting to send in daemon mode (default: 20m).
- `--port` - VSock port (default: 818).
- `--repo-cpe-url` - URL for the repository to CPE mapping.
- `--timeout` - VSock client timeout when sending index reports.
- `--verbose` - Prints the index reports to stdout.

## How it works

1. Scans filesystem for `rpm`/`dnf` package databases.
2. Pulls repo-to-CPE mappings from Red Hat. Network connection to the public Internet or to Sensor is required.
3. Creates protobuf index report.
4. Sends report to host via vsock.

The host receives these reports and forwards them to StackRox Central for vulnerability analysis.

## Deployment

### Using Quadlet (Recommended for RHEL VMs)

For RHEL 9 VMs, use Podman Quadlet to run roxagent as a periodic systemd service.
See [quadlet/README.md](quadlet/README.md) for detailed instructions.

```bash
cd quadlet
./install.sh              # Install locally
./install.sh user@host    # Install on remote VM
```

### Building from Source

```bash
go build -o roxagent .

# For Linux VMs
GOOS=linux GOARCH=amd64 go build -o roxagent-linux .
```

## Troubleshooting

### Can't connect to host

- Check if vsock is enabled in the VM.
- Verify the port isn't in use.
- Make sure vsock kernel modules are loaded.

### No packages found

- Check `--host-path` points to the right place.
- Verify `rpm`/`dnf` databases exist and are readable.
- Use `--verbose` to examine the index report and compare with the content from `rpm`/`dnf` databases.

### Scan failures

- Check internet access for repo-to-CPE downloads.
- Look at logs for specific errors.
