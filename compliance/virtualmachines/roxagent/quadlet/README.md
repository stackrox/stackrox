# Quadlet Deployment for roxagent

Deploy roxagent as a periodic systemd service on RHEL VMs using Podman Quadlet.

## Overview

This deployment uses [Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) to run roxagent from a container image as a systemd service. The agent runs hourly, scans installed packages, and reports them to StackRox via vsock.

### Components

| File | Description |
|------|-------------|
| `roxagent.container` | Quadlet container unit that runs roxagent |
| `roxagent.timer` | Systemd timer that triggers hourly scans |
| `roxagent-prep.service` | Prepares RPM database for scanning |
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
Image=quay.io/stackrox-io/main:4.10.0
```

Use the same version as your StackRox Central deployment.

### 2. Install the Units

**Local installation:**

```bash
./install.sh
```

**Remote installation via SSH:**

```bash
./install.sh user@hostname
./install.sh user@hostname 2222  # Custom SSH port
```

### 3. Verify Installation

```bash
# Check timer status
sudo systemctl list-timers roxagent.timer

# Run immediately
sudo systemctl start roxagent.service

# View logs
sudo journalctl -u roxagent.service -f
```

## How It Works

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ RHEL VM                                                     │
│  ┌─────────────────┐                                        │
│  │ roxagent.timer  │ ──(hourly)──▶ roxagent.service         │
│  └─────────────────┘                      │                 │
│                                           ▼                 │
│  ┌─────────────────┐         ┌────────────────────────┐     │
│  │ roxagent-prep   │ ──────▶ │ roxagent container     │     │
│  │ (copy RPM db)   │         │ - scans /host/var/lib/ │     │
│  └─────────────────┘         │ - sends via vsock      │     │
│                              └───────────┬────────────┘     │
└──────────────────────────────────────────┼──────────────────┘
                                           │ vsock
┌──────────────────────────────────────────┼─────────────────┐
│ Kubernetes Host                          ▼                 │
│  ┌────────────────────────────────────────────┐            │
│  │ collector pod (compliance container)       │            │
│  │ - receives vsock connections               │            │
│  │ - forwards to Sensor                       │            │
│  └─────────────────────┬──────────────────────┘            │
│                        │ gRPC                              │
│                        ▼                                   │
│  ┌────────────────────────────────────────────┐            │
│  │ Sensor ──▶ Central                         │            │
│  └────────────────────────────────────────────┘            │
└────────────────────────────────────────────────────────────┘
```

### Why Copy the RPM Database?

The `roxagent-prep.service` copies `/var/lib/rpm` to `/tmp/roxagent-rpm` before each scan. This is required because:

1. **SQLite WAL Mode**: RHEL 9 and 10 use SQLite for the RPM database. SQLite's Write-Ahead Logging (WAL) requires write access even for read-only queries. RHEL 8 uses BerkeleyDB, which also benefits from copying.

2. **Safety**: Copying protects the host's RPM database from any potential issues during scanning.

3. **Consistency**: The copy provides a point-in-time snapshot, avoiding conflicts if packages are installed during the scan.

## Configuration

### Scan Interval

Edit `roxagent.timer` to change the scan frequency:

```ini
[Timer]
OnBootSec=5min      # First scan after boot
OnUnitActiveSec=1h  # Subsequent scans (change to 30m, 2h, etc.)
```

### Container Options

Edit `roxagent.container` to customize:

```ini
# Add verbose output
Exec=--verbose --host-path /host

# Change vsock port (must match StackRox config)
Exec=--host-path /host --port 2048
```

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
2. Check compliance container logs in the collector pod
3. Verify Sensor can reach Central

## Uninstallation

```bash
sudo systemctl disable --now roxagent.timer
sudo rm /etc/containers/systemd/roxagent.container
sudo rm /etc/systemd/system/roxagent.timer
sudo rm /etc/systemd/system/roxagent-prep.service
sudo systemctl daemon-reload
```
