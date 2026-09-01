# roxagent

roxagent runs inside KubeVirt VMs. It scans installed packages for vulnerabilities and serves index reports to Sensor over VSOCK.

Sensor's VM scraper dials the agent, pulls a cached report, and forwards it to Central for vulnerability matching. Start the agent with `roxagent serve`.

The sources live under `compliance/` because the agent reuses the Scanner V4 node indexer for RPM/DNF package databases.

## What it does

1. Listens on a VSOCK port with mandatory mTLS. Startup does not wait for a mapping or a first scan.
2. Obtains a repository-to-CPE mapping (Sensor push over VSOCK by default, or a download URL). Until one is ready, GetReport returns `MAPPING_REQUIRED` and no scan runs.
3. Scans the VM for installed packages and caches the index report with a content-hash token.
4. On each Sensor connection, handles a framed `VMServiceRequest` / `VMServiceResponse` (for example `get_report` and `sync_repo_cpe_mapping`).
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
| `--repo-cpe-url` | empty (Sensor-managed) | Mapping download URL. Empty means Sensor pushes the mapping over VSOCK. |
| `--repo-cpe-bundled-path` | empty | Optional seed mapping file for Sensor-managed agents. Ignored when `--repo-cpe-url` is set. |
| `--rescan-interval` | `4h` | Must be between `5m` and `168h` (7d) |
| `--ca-fetch-timeout` | `10s` | Timeout for one KubeVirt CA fetch over VSOCK |
| `--conn-deadline` | `30s` | Max time for one connection's TLS handshake plus request/response (`5s`-`5m`) |

## How it works

1. **Mapping file:** Every scan reads a local cache file kept fresh by a mapping provider. With `--repo-cpe-url` empty (the default), Sensor pushes the mapping over VSOCK via `sync_repo_cpe_mapping`. `--repo-cpe-bundled-path` is an optional seed for that Sensor-managed path when there is no cache yet. With a URL set, the agent downloads in the background and never accepts a Sensor push or a bundled seed: a URL that never succeeds stays not-ready rather than looking configured. A failed later URL refresh keeps the last-good cache from a previous successful fetch of that URL.

   A URL-backed agent needs outbound HTTPS to that host (or a proxy). Isolated networks either rely on Sensor (optionally with a bundled seed), or host a copy of `repository-to-cpe.json` the VM can reach and pass `--repo-cpe-url`.
2. **Listen first:** VSOCK comes up immediately. The first scan runs when a mapping is ready (cached from a previous run, a bundled seed, a successful URL fetch, or a Sensor push), not at process start.
3. **Serving reports:** Sensor connects and the agent serves the cached report. If Sensor already has the current content-hash token, the agent omits the payload. Sensor can still force a full report by sending an empty token (first request, NACK retry, or the 4h refresh). With no mapping yet, the response is `MAPPING_REQUIRED`.
4. **Rescan:** On `--rescan-interval`, or sooner when the mapping changes, re-indexes the rpm/dnf database and swaps the cache. The token stays the same when the report and facts are unchanged.

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
