# Quadlet Deployment for roxagent (Pull Mode)

Deploy roxagent as a long-running VSOCK server on RHEL VMs using Podman Quadlet.

## Overview

This deployment uses [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) to run `roxagent serve` from a container image as a systemd service. The agent starts once, scans installed packages, caches the report, and listens on a VSOCK port for Sensor to pull results on demand. Periodic rescans happen internally (default: every 4 hours).

### Components

| File | Description |
|------|-------------|
| `roxagent.container` | Quadlet container unit that runs `roxagent serve` |
| `roxagent-prep.service` | Copies RPM database to a writable location |
| `roxagent-tmpfiles.conf` | Recreates `/run/lock/roxagent` on boot |
| `install.sh` | Installation script for local or remote deployment |

## Prerequisites

* RHEL 8, 9, or 10 VM running on KubeVirt with vsock enabled
* Podman installed (`dnf install -y podman`)
* StackRox deployed with VM scanning enabled (`ROX_VIRTUAL_MACHINES=true`)
* Network access to pull the StackRox main image

## Installation

### 1. Configure the Image Tag

Edit `roxagent.container` and set the correct image tag:

```ini
Image=quay.io/stackrox-io/main:4.11.0
```

Use the same version as your StackRox Central deployment.

### 2. Install the Units

**Local installation:**

```bash
./install.sh
```

**Remote installation via SSH:**

```bash
./install.sh --ssh user@hostname
./install.sh --ssh user@hostname 2222  # Custom SSH port
```

**Remote installation via virtctl:**

```bash
./install.sh --virtctl -n openshift-cnv cloud-user@vmi/rhel10-1
```

### 3. Verify Installation

```bash
# Check service status
sudo systemctl status roxagent.service

# View logs
sudo journalctl -u roxagent.service -f

# Restart after config change
sudo systemctl restart roxagent.service
```

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│ RHEL VM                                                     │
│  ┌─────────────────┐         ┌────────────────────────┐     │
│  │ roxagent-prep   │ ──────▶ │ roxagent container     │     │
│  │ (copy RPM db)   │         │ - roxagent serve       │     │
│  └─────────────────┘         │ - listens on VSOCK     │     │
│                              │ - rescans every 4h     │     │
│                              └───────────┬────────────┘     │
└──────────────────────────────────────────┼──────────────────┘
                                           │ vsock (pull)
┌──────────────────────────────────────────┼─────────────────┐
│ Kubernetes Host                          ▼                 │
│  ┌────────────────────────────────────────────┐            │
│  │ Sensor                                     │            │
│  │ - pulls reports via VSOCK on demand        │            │
│  │ - forwards to Central                      │            │
│  └────────────────────────────────────────────┘            │
└────────────────────────────────────────────────────────────┘
```

### Why Copy the RPM Database?

The `roxagent-prep.service` copies `/var/lib/rpm` to `/tmp/roxagent-rpm` before the container starts. This is required because SQLite WAL mode (used by RHEL 9+) requires write access even for read-only queries. The copy also provides safety and a consistent point-in-time snapshot.

In pull mode, the copy happens once at container start. A container restart (manual or via `Restart=on-failure`) triggers a fresh copy.

## Configuration

### Rescan Interval

The agent rescans internally. To change the interval, edit the `Exec=` line in `roxagent.container`:

```ini
Exec=serve --host-path /host --rescan-interval 2h
```

### VSOCK Port

```ini
Exec=serve --host-path /host --port 2048
```

The port must match the StackRox Sensor configuration.

## Troubleshooting

### No packages found

Check if the RPM database copy succeeded:

```bash
ls -la /tmp/roxagent-rpm/
sudo journalctl -u roxagent-prep.service
```

### vsock connection failed

Verify vsock is enabled in the VM:

```bash
ls -la /dev/vsock
lsmod | grep vsock
```

### Container fails to start

Check Quadlet generation:

```bash
/usr/libexec/podman/quadlet --dryrun
sudo journalctl -u roxagent.service
```

### VM not appearing in Central

1. Verify `ROX_VIRTUAL_MACHINES=true` is set on Central and Sensor
2. Check Sensor logs for VSOCK scraper activity
3. Verify Sensor can reach Central

## Uninstallation

```bash
sudo systemctl disable --now roxagent.service
sudo rm /etc/containers/systemd/roxagent.container
sudo rm /etc/systemd/system/roxagent-prep.service
sudo rm /etc/tmpfiles.d/roxagent.conf
sudo systemctl daemon-reload
```
