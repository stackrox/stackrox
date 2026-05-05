#!/bin/bash
# Install roxagent Quadlet units on a RHEL VM
#
# Usage:
#   ./install.sh                    # Install locally
#   ./install.sh user@host          # Install on remote host via SSH
#   ./install.sh user@host 2222     # Install on remote host with custom SSH port

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Host paths that may not exist on all RHEL versions (e.g. DNF paths on
# yum-only RHEL 8). Volume= lines referencing these are stripped when the
# source path is absent.
OPTIONAL_HOST_PATHS=(
    /etc/yum.repos.d
    /etc/yum/repos.d
    /etc/distro.repos.d
    /etc/redhat-release
    /etc/system-release-cpe
    /var/cache/dnf
    /var/lib/dnf
)

# Produce a filtered roxagent.container on stdout, removing Volume= lines
# whose host source path does not exist on this machine.
filter_container_file() {
    local file="${1}"
    local pattern=""
    for p in "${OPTIONAL_HOST_PATHS[@]}"; do
        if [ ! -e "${p}" ]; then
            echo "Stripping mount for missing path: ${p}" >&2
            pattern="${pattern:+${pattern}|}Volume=${p}[:/]"
        fi
    done
    if [ -z "${pattern}" ]; then
        cat "${file}"
    else
        grep -Ev "${pattern}" "${file}"
    fi
}

install_locally() {
    echo "Installing Quadlet units locally..."

    # Quadlet container file (strip mounts for paths missing on this host)
    sudo mkdir -p /etc/containers/systemd/
    filter_container_file "${SCRIPT_DIR}/roxagent.container" \
        | sudo tee /etc/containers/systemd/roxagent.container >/dev/null
    # restorecon resets SELinux labels so systemd/podman can read the new files.
    sudo restorecon -Rv /etc/containers/systemd/ 2>/dev/null || true

    # Timer and prep service go in standard systemd directory
    sudo cp "${SCRIPT_DIR}/roxagent.timer" /etc/systemd/system/
    sudo cp "${SCRIPT_DIR}/roxagent-prep.service" /etc/systemd/system/
    sudo restorecon -Rv /etc/systemd/system/roxagent.timer /etc/systemd/system/roxagent-prep.service 2>/dev/null || true

    # Recreate the lock directory on every boot since /run is tmpfs.
    sudo mkdir -p /etc/tmpfiles.d/
    sudo cp "${SCRIPT_DIR}/roxagent-tmpfiles.conf" /etc/tmpfiles.d/roxagent.conf
    sudo restorecon -Rv /etc/tmpfiles.d/roxagent.conf 2>/dev/null || true
    # systemd-tmpfiles --create applies the rule now (creates /run/lock/roxagent immediately).
    sudo systemd-tmpfiles --create /etc/tmpfiles.d/roxagent.conf

    echo "Reloading systemd..."
    sudo systemctl daemon-reload

    echo "Enabling and starting timer..."
    sudo systemctl enable --now roxagent.timer

    echo "Status:"
    sudo systemctl list-timers roxagent.timer
}

install_remote() {
    local REMOTE_HOST="${1}"
    local SSH_PORT="${2:-22}"

    echo "Installing Quadlet units on ${REMOTE_HOST} (port ${SSH_PORT})..."

    # Copy files
    scp -P "${SSH_PORT}" "${SCRIPT_DIR}/roxagent.container" "${REMOTE_HOST}:/tmp/"
    scp -P "${SSH_PORT}" "${SCRIPT_DIR}/roxagent.timer" "${REMOTE_HOST}:/tmp/"
    scp -P "${SSH_PORT}" "${SCRIPT_DIR}/roxagent-prep.service" "${REMOTE_HOST}:/tmp/"
    scp -P "${SSH_PORT}" "${SCRIPT_DIR}/roxagent-tmpfiles.conf" "${REMOTE_HOST}:/tmp/"

    # Install on remote — filter container file for missing optional paths
    ssh -p "${SSH_PORT}" "${REMOTE_HOST}" << 'EOF'
        set -euo pipefail

        OPTIONAL_HOST_PATHS=(
            /etc/yum.repos.d
            /etc/yum/repos.d
            /etc/distro.repos.d
            /etc/redhat-release
            /etc/system-release-cpe
            /var/cache/dnf
            /var/lib/dnf
        )

        # Strip Volume= lines for host paths that don't exist on this machine
        pattern=""
        for p in "${OPTIONAL_HOST_PATHS[@]}"; do
            if [ ! -e "${p}" ]; then
                echo "  Stripping mount for missing path: ${p}"
                pattern="${pattern:+${pattern}|}Volume=${p}[:/]"
            fi
        done
        if [ -n "${pattern}" ]; then
            grep -Ev "${pattern}" /tmp/roxagent.container > /tmp/roxagent.container.filtered
            mv /tmp/roxagent.container.filtered /tmp/roxagent.container
        fi

        # Quadlet container file
        sudo mkdir -p /etc/containers/systemd/
        sudo mv /tmp/roxagent.container /etc/containers/systemd/
        # restorecon resets SELinux labels so systemd/podman can read the new files.
        sudo restorecon -Rv /etc/containers/systemd/ 2>/dev/null || true

        # Timer and prep service go in standard systemd directory
        sudo mv /tmp/roxagent.timer /etc/systemd/system/
        sudo mv /tmp/roxagent-prep.service /etc/systemd/system/
        sudo restorecon -Rv /etc/systemd/system/roxagent.timer /etc/systemd/system/roxagent-prep.service 2>/dev/null || true

        # Recreate the lock directory on every boot since /run is tmpfs.
        sudo mkdir -p /etc/tmpfiles.d/
        sudo mv /tmp/roxagent-tmpfiles.conf /etc/tmpfiles.d/roxagent.conf
        sudo restorecon -Rv /etc/tmpfiles.d/roxagent.conf 2>/dev/null || true
        # systemd-tmpfiles --create applies the rule now (creates /run/lock/roxagent immediately).
        sudo systemd-tmpfiles --create /etc/tmpfiles.d/roxagent.conf

        echo "Reloading systemd..."
        sudo systemctl daemon-reload

        echo "Enabling and starting timer..."
        sudo systemctl enable --now roxagent.timer

        echo "Status:"
        sudo systemctl list-timers roxagent.timer
EOF
}

# Main
if [ "${#}" -eq 0 ]; then
    install_locally
else
    install_remote "${1}" "${2:-22}"
fi

echo ""
echo "Done! The roxagent will run hourly."
echo ""
echo "To run immediately:  sudo systemctl start roxagent.service"
echo "To view logs:        sudo journalctl -u roxagent.service -f"
echo "To check timer:      sudo systemctl list-timers roxagent.timer"
