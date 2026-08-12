# Quadlet Deployment for roxagent

Run roxagent on RHEL VMs with Podman Quadlet: a systemd-managed container that serves index reports to Sensor over VSOCK.

## Overview

[Podman Quadlet](https://docs.podman.io/en/latest/markdown/podman-systemd.unit.5.html) turns `roxagent.container` into a systemd unit that runs `roxagent serve`. The agent scans installed packages, caches the report, and listens on a VSOCK port. Sensor dials in when it needs a report. The agent rescans on its own schedule (default: every 4 hours).

### Components

| File | Description |
|------|-------------|
| `roxagent.container` | Quadlet container unit that runs `roxagent serve` |
| `install.sh` | Installation script for local or remote deployment |

## Prerequisites

* RHEL 8, 9, or 10 VM running on KubeVirt with VSOCK enabled
* Podman installed (`dnf install -y podman`)
* StackRox deployed with VM scanning enabled (`ROX_VIRTUAL_MACHINES=true`)
* Network access to pull the StackRox main image

## Installation

### 1. Configure the Image Tag

Edit `roxagent.container` and set the image tag to match your StackRox Secured Cluster deployments:

```ini
Image=quay.io/stackrox-io/main:4.11.0
```

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

# After editing roxagent.container, reload then restart
sudo systemctl daemon-reload
sudo systemctl restart roxagent.service
```

## How It Works

```
┌─────────────────────────────────────────────────────────────┐
│ RHEL VM                                                     │
│  ┌────────────────────────┐                                 │
│  │ roxagent container     │                                 │
│  │ - mounts live RPM/DNF  │                                 │
│  │   DBs read-only        │                                 │
│  │ - listens on VSOCK     │                                 │
│  │ - rescans every 4h     │                                 │
│  └───────────┬────────────┘                                 │
└──────────────┼──────────────────────────────────────────────┘
               ▲ vsock
┌──────────────┼─────────────────────────────────────────────┐
│ Kubernetes Host                          │                 │
│  ┌────────────────────────────────────────────┐            │
│  │ Sensor                                     │            │
│  │ - connects over VSOCK and fetches reports  │            │
│  │ - forwards to Central                      │            │
│  └────────────────────────────────────────────┘            │
└────────────────────────────────────────────────────────────┘
```

### Why Mount the RPM Database Read-Only?

The container bind-mounts the live host `/var/lib/rpm` (and DNF history paths) read-only under `/host`. Claircore copies each SQLite DB to a temp spool before opening it with `query_only`, so WAL sidecars are created next to the spool inside the container, never on the host. That keeps package inventory fresh across rescans without copying the DB at start, and without giving the agent a writable host RPM directory.

Optional DNF paths (`/var/lib/dnf`, `/usr/lib/sysimage/libdnf5`, `/var/cache/dnf`, repo dirs) are stripped by `install.sh` when missing on the guest.

## Configuration

### Rescan Interval

To change how often the agent rescans, edit the `Exec=` line in `roxagent.container`:

```ini
Exec=serve --host-path /host --rescan-interval 2h
```

### VSOCK Port

```ini
Exec=serve --host-path /host --port 2048
```

The port must match the StackRox Sensor configuration.

## Troubleshooting

### No Packages Found

Confirm the live RPM database is mounted and readable in the container:

```bash
ls -la /var/lib/rpm/
sudo journalctl -u roxagent.service
```

Make sure that your VM guest OS is activated and has executed at least a single DNF transaction (e.g., dnf install, dnf update).

### VSOCK Connection Failed

Verify VSOCK is enabled in the VM:

```bash
ls -la /dev/vsock
lsmod | grep vsock
```

### Container Fails to Start

Check Quadlet generation:

```bash
/usr/libexec/podman/quadlet --dryrun
sudo journalctl -u roxagent.service
```

### VM Not Appearing in Central

1. Verify `ROX_VIRTUAL_MACHINES=true` is set on Central and Sensor
2. Check Sensor logs for VSOCK scraper activity
3. Verify Sensor can reach Central

## Uninstallation

```bash
sudo systemctl stop roxagent.service
sudo rm /etc/containers/systemd/roxagent.container
sudo systemctl daemon-reload
```

Quadlet units are generated under `/run/systemd/generator/`, so `systemctl enable` / `disable` do not apply. Boot start comes from `[Install] WantedBy=multi-user.target` in `roxagent.container`; removing that file and reloading drops the service.
