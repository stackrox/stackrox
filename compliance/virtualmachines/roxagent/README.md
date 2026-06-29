# roxagent

Runs inside KubeVirt VMs to scan for vulnerabilities and serve reports to Sensor
via VSOCK (pull mode). While co-located under `compliance/`, the agent reuses
compliance node-scanning code for package indexing — it is not part of the
Compliance Operator feature.

## What it does

1. Scans the VM for installed packages (`rpm`/`dnf` databases) using the same
   Scanner V4 indexer as node scanning.
2. Caches the scan report in memory with a generation counter.
3. Listens on a VSOCK port for incoming connections from Sensor.
4. When Sensor connects, serves the cached report via the `VMServiceRequest` /
   `VMServiceResponse` framed protobuf protocol.
5. Periodically rescans to pick up package changes.

Sensor pulls reports from all running VMs on a timer and forwards them to
Central for vulnerability matching.

## Usage

```bash
# Start pull-mode server (production mode)
sudo ./roxagent serve

# Custom settings
sudo ./roxagent serve --port 818 --host-path / --rescan-interval 4h
```

## Flags (`serve` subcommand)

- `--port` — VSOCK port to listen on (default: 818).
- `--host-path` — Root filesystem path for package indexing (default: /).
- `--repo-cpe-url` — URL for the repository-to-CPE mapping file.
- `--rescan-interval` — Interval between periodic rescans (default: 4h).

## How it works

1. Performs an initial scan of the VM filesystem for package databases.
2. Fetches repo-to-CPE mappings from Red Hat (requires network access).
3. Starts a VSOCK listener with optional mTLS (KubeVirt CA).
4. On each Sensor connection: reads a `VMServiceRequest`, dispatches by method,
   returns the cached `VMServiceResponse` with the index report.
5. On rescan timer: re-indexes the filesystem and atomically swaps the cached
   report, incrementing the generation counter.

### TLS

When running inside a KubeVirt VM with TLS enabled:
- roxagent fetches the KubeVirt CA from the host (VSOCK CID 2, port 1).
- Connections from Sensor (via virt-handler) present a client cert signed by
  the KubeVirt CA, which roxagent validates.
- roxagent uses a self-signed server cert (virt-handler does not validate it).
- The CA is refreshed hourly to support rotation.

If the KubeVirt CA is unavailable, roxagent falls back to plaintext VSOCK
(RBAC on the KubeVirt subresource still gates access).

## Deployment

### Native systemd service (CI / dev)

The CI script `scripts/ci/add-vms/install-agent-native.sh` builds roxagent,
copies it into the VM via `virtctl scp`, and enables a systemd service:

```bash
# roxagent-serve.service runs: /usr/local/bin/roxagent serve
```

### Quadlet (RHEL VMs)

See [quadlet/README.md](quadlet/README.md) for Podman Quadlet deployment.
Note: Quadlet units may still reference the old push-mode entrypoint and need
updating for pull mode.

### Building from source

```bash
# For the current platform
go build -o roxagent .

# Cross-compile for Linux VMs
GOOS=linux GOARCH=amd64 go build -o roxagent .
```

## Troubleshooting

### Can't connect / dial failures from Sensor

- Verify VSOCK is enabled on the VMI spec (`spec.domain.devices.autoattachVSOCK`).
- Check that the VSOCK port isn't in use by another process inside the VM.
- Ensure Sensor has RBAC for `virtualmachineinstances/vsock` on `subresources.kubevirt.io`.

### No packages found (zero-package reports)

- Check `--host-path` points to the correct root filesystem.
- Verify `rpm`/`dnf` databases exist and are readable.
- Check Sensor logs for `reportcheck` warnings.

### TLS handshake failures

- Verify KubeVirt has TLS enabled (check virt-handler logs).
- Check that roxagent can reach CID 2 port 1 (KubeVirt CA service).
- Look for "Rejected plaintext connection" in roxagent logs (Sensor not using TLS).
