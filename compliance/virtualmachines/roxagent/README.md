# roxagent

roxagent runs inside KubeVirt VMs. It scans installed packages for vulnerabilities and serves index reports to Sensor over VSOCK.

Sensor's VM scraper dials the agent, pulls a cached report, and forwards it to Central for vulnerability matching. Start the agent with `roxagent serve`.

The sources live under `compliance/` because the agent reuses the Scanner V4 node indexer for RPM/DNF package databases.

## What it does

1. Fetches the repository-to-CPE mapping file (required before the first scan).
2. Scans the VM for installed packages and caches the index report with a content-hash token.
3. Listens on a VSOCK port with mandatory mTLS.
4. On each Sensor connection, handles a framed `VMServiceRequest` / `VMServiceResponse` (for example `get_report`).
5. Periodically rescans and atomically swaps the cached report.

## Usage

```bash
sudo ./roxagent serve

# Example with common overrides
sudo ./roxagent serve --port 818 --host-path / --rescan-interval 4h
```

## Flags (`serve`)

| Flag | Default | Notes |
|------|---------|-------|
| `--port` | `818` | VSOCK listen port |
| `--host-path` | `/` | Root filesystem path for package indexing |
| `--repo-cpe-url` | Red Hat security data URL | Repository-to-CPE mapping download URL |
| `--rescan-interval` | `4h` | Must be between `5m` and `168h` (7d) |
| `--ca-fetch-timeout` | `10s` | Timeout for one KubeVirt CA fetch over VSOCK |
| `--conn-deadline` | `30s` | Max time for one connection's TLS handshake plus request/response (`5s`-`5m`) |

## How it works

1. **Mapping file:** On startup the agent downloads the repository-to-CPE mapping into a local cache and blocks until that succeeds. Every scan reads from that file. If a later refresh fails, the agent keeps the last good cache and continues scanning.

   The default `--repo-cpe-url` is `https://security.access.redhat.com/data/metrics/repository-to-cpe.json`, so the VM needs outbound HTTPS to that host (or a proxy that can reach it). On isolated networks, host a copy of `repository-to-cpe.json` somewhere the VM can reach and point `--repo-cpe-url` at that URL.
2. **Initial scan:** Indexes `--host-path`, stores the report and discovered VM facts in an in-memory cache, then starts the VSOCK server and the rescan loop.
3. **Serving reports:** Sensor connects and the agent serves the cached report. If Sensor already has the current content-hash token, the agent omits the payload. Sensor can still force a full report by sending an empty token (first request, NACK retry, or the 4h refresh).
4. **Rescan:** On `--rescan-interval`, re-indexes the rpm/dnf database and swaps the cache. The token stays the same when the report and facts are unchanged.

### TLS (mandatory)

Sensor always dials with TLS. The agent always requires TLS; there is no plaintext mode.

- On each handshake, the agent fetches the KubeVirt CA from the host (VSOCK CID 2, port 1) via virt-handler's `System.CABundle` RPC. In namespace-isolated VSOCK mode that CA service is only up for the duration of the dial/handshake, so the fetch runs inside the handshake.
- If a fetch fails but a CA was cached earlier, the agent reuses the cache. If there is no cache yet, the handshake fails.
- Sensor (via virt-handler) presents a client cert signed by the KubeVirt CA; the agent verifies it.
- The agent uses a self-signed server cert (virt-handler does not verify the server cert).

## Deployment

### Native systemd (CI / GitHub Action)

CI and the [Add VMs to Cluster](../../../.github/workflows/add-vms-to-cluster.yml) workflow install through `scripts/ci/add-vms/`:

1. Workflow → composite action `scripts/ci/add-vms`
2. `add-vms.sh` deploys VMs, then calls `install-agent-native.sh`
3. `install-agent-native.sh` cross-compiles roxagent, copies it with `virtctl scp`, and installs two units:
   - `roxagent-prep.service` builds the `/tmp/roxroot` mount-point skeleton and copies `/var/lib/rpm` to `/tmp/roxagent-rpm` (writable, for SQLite WAL - see [quadlet/README.md](quadlet/README.md#why-copy-the-rpm-database))
   - `roxagent-serve.service` bind-mounts the real host paths into `/tmp/roxroot` and runs:

```bash
ExecStart=/usr/local/bin/roxagent serve --port 818 --host-path /tmp/roxroot
```

Verification waits until `roxagent-serve.service` is enabled and active. That is the same path developers get when they run the workflow against an infra cluster.

### Quadlet (RHEL VMs)

See [quadlet/README.md](quadlet/README.md) for Podman Quadlet deployment (`Exec=serve`, systemd unit `roxagent.service`).

### Building from source

```bash
go build -o roxagent .

GOOS=linux GOARCH=amd64 go build -o roxagent .
```

## Troubleshooting

### Can't connect / dial failures from Sensor

- Confirm VSOCK is enabled on the VMI (`spec.domain.devices.autoattachVSOCK`).
- Confirm the VSOCK port is free inside the VM and matches Sensor's config.
- Confirm Sensor has RBAC for `virtualmachineinstances/vsock` on `subresources.kubevirt.io`.

### No packages found (zero-package reports)

- Confirm `--host-path` points at the real root (native CI uses `/tmp/roxroot` with bind mounts).
- Confirm `rpm`/`dnf` databases exist and are readable under that path.
- Check Sensor logs for `reportcheck` warnings.

### TLS handshake failures

- Confirm virt-handler is serving the CA on CID 2 port 1 during the dial.
- On first connect with a cold CA cache, a failed CA fetch fails the handshake (no plaintext fallback).
- Look for `Rejected plaintext connection` if the peer is not speaking TLS.
- Tune `--ca-fetch-timeout` / `--conn-deadline` if handshakes time out under load.
