#!/bin/bash
# Install roxagent Quadlet units on a RHEL VM
#
# Usage:
#   ./install.sh                                                      # Install locally
#   ./install.sh user@host                                            # SSH (port 22)
#   ./install.sh user@host 2222                                       # SSH with custom port
#   ./install.sh virtctl -n openshift-cnv cloud-user@vmi/rhel10-1     # Via virtctl

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

# --- Transport abstraction ---------------------------------------------------
# Each transport stores state here and uses the shared helpers:
#   remote_copy <local-file> <remote-dest>   — copy a file to the target
#   remote_exec                              — run a script from stdin on target

TRANSPORT_KIND=""
SSH_HOST=""
SSH_PORT=""
VIRTCTL_TARGET=""
VIRTCTL_FLAGS=()

remote_copy() {
    case "${TRANSPORT_KIND}" in
        ssh)
            scp -P "${SSH_PORT}" "$1" "${SSH_HOST}:$2"
            ;;
        virtctl)
            virtctl scp "${VIRTCTL_FLAGS[@]}" "$1" "${VIRTCTL_TARGET}:$2"
            ;;
        *)
            echo "remote_copy called without a remote transport" >&2
            return 1
            ;;
    esac
}

remote_exec() {
    case "${TRANSPORT_KIND}" in
        ssh)
            ssh -p "${SSH_PORT}" "${SSH_HOST}" bash -s
            ;;
        virtctl)
            virtctl ssh "${VIRTCTL_FLAGS[@]}" "${VIRTCTL_TARGET}" --command 'bash -s'
            ;;
        *)
            echo "remote_exec called without a remote transport" >&2
            return 1
            ;;
    esac
}

setup_transport_local() {
    TRANSPORT_KIND="local"
}

setup_transport_ssh() {
    TRANSPORT_KIND="ssh"
    SSH_HOST="$1"
    SSH_PORT="${2:-22}"
}

setup_transport_virtctl() {
    # Expects virtctl flags + target, e.g.:
    #   -n openshift-cnv cloud-user@vmi/rhel10-1
    # The last positional arg is the target; everything else are flags.
    TRANSPORT_KIND="virtctl"
    VIRTCTL_TARGET="${*: -1}"
    if (( $# > 1 )); then
        VIRTCTL_FLAGS=("${@:1:$#-1}")
    else
        VIRTCTL_FLAGS=()
    fi
}

# --- Install logic (shared) --------------------------------------------------

REMOTE_INSTALL_SCRIPT=$(cat << 'SCRIPT'
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
    if [ ! -e "$p" ]; then
        echo "  Stripping mount for missing path: $p"
        pattern="${pattern:+${pattern}|}Volume=${p}[:/]"
    fi
done
if [ -n "$pattern" ]; then
    grep -Ev "$pattern" /tmp/roxagent.container > /tmp/roxagent.container.filtered
    mv /tmp/roxagent.container.filtered /tmp/roxagent.container
fi

# Quadlet container file
sudo mkdir -p /etc/containers/systemd/
sudo mv /tmp/roxagent.container /etc/containers/systemd/
sudo restorecon -Rv /etc/containers/systemd/ 2>/dev/null || true

# Timer and prep service
sudo mv /tmp/roxagent.timer /etc/systemd/system/
sudo mv /tmp/roxagent-prep.service /etc/systemd/system/
sudo restorecon -Rv /etc/systemd/system/roxagent.timer /etc/systemd/system/roxagent-prep.service 2>/dev/null || true

echo "Reloading systemd..."
sudo systemctl daemon-reload

echo "Enabling and starting timer..."
sudo systemctl enable --now roxagent.timer

echo "Status:"
sudo systemctl list-timers roxagent.timer
SCRIPT
)

install_local() {
    echo "Installing Quadlet units locally..."

    # Filter container file for missing optional paths
    local filtered
    filtered=$(filter_container_file "${SCRIPT_DIR}/roxagent.container")

    sudo mkdir -p /etc/containers/systemd/
    echo "$filtered" | sudo tee /etc/containers/systemd/roxagent.container >/dev/null
    sudo restorecon -Rv /etc/containers/systemd/ 2>/dev/null || true

    sudo cp "${SCRIPT_DIR}/roxagent.timer" /etc/systemd/system/
    sudo cp "${SCRIPT_DIR}/roxagent-prep.service" /etc/systemd/system/
    sudo restorecon -Rv /etc/systemd/system/roxagent.timer /etc/systemd/system/roxagent-prep.service 2>/dev/null || true

    echo "Reloading systemd..."
    sudo systemctl daemon-reload

    echo "Enabling and starting timer..."
    sudo systemctl enable --now roxagent.timer

    echo "Status:"
    sudo systemctl list-timers roxagent.timer
}

install_remote() {
    echo "Copying files to target..."
    remote_copy "${SCRIPT_DIR}/roxagent.container" /tmp/
    remote_copy "${SCRIPT_DIR}/roxagent.timer" /tmp/
    remote_copy "${SCRIPT_DIR}/roxagent-prep.service" /tmp/

    echo "Running install on target..."
    echo "$REMOTE_INSTALL_SCRIPT" | remote_exec
}

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

# --- Main ---------------------------------------------------------------------

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
if [ $# -eq 0 ]; then
    setup_transport_local
    install_local
elif [[ "$1" == "virtctl" ]]; then
    shift
    setup_transport_virtctl "$@"
    install_remote
else
    setup_transport_ssh "$1" "${2:-22}"
    install_remote
fi

echo ""
echo "Done! The roxagent will run hourly."
echo ""
echo "To run immediately:  sudo systemctl start roxagent.service"
echo "To view logs:        sudo journalctl -u roxagent.service -f"
echo "To check timer:      sudo systemctl list-timers roxagent.timer"
