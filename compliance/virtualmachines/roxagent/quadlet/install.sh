#!/bin/bash
# Install roxagent Quadlet units on a RHEL VM
#
# Usage:
#   ./install.sh                                                      # Install locally
#   ./install.sh -n openshift-cnv cloud-user@vmi/rhel10-1             # Via virtctl (default remote)
#   ./install.sh --ssh user@host                                      # SSH (port 22)
#   ./install.sh --ssh user@host 2222                                 # SSH with custom port

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

STAGED_INSTALL_FILES=(
    install.sh
    roxagent.container
    roxagent.timer
    roxagent-prep.service
    roxagent-tmpfiles.conf
)

TRANSPORT_KIND=""
SSH_HOST=""
SSH_PORT=""
VIRTCTL_TARGET=""
VIRTCTL_FLAGS=()

main() {
    if [ $# -eq 0 ]; then
        setup_transport_local
        install_local
    elif [ "$1" = "--stage-dir" ]; then
        if [ $# -ne 2 ]; then
            echo "usage: $0 --stage-dir <dir>" >&2
            exit 1
        fi
        install_from_stage_dir "$2"
        return
    elif [ "$1" = "--ssh" ]; then
        shift
        if [ $# -lt 1 ]; then
            usage
            exit 1
        fi
        setup_transport_ssh "$1" "${2:-22}"
        install_remote
    else
        setup_transport_virtctl "$@"
        install_remote
    fi

    echo ""
    echo "Done! The roxagent will run periodically."
    echo ""
    echo "To run immediately:  sudo systemctl start roxagent.service"
    echo "To view logs:        sudo journalctl -u roxagent.service -f"
    echo "To check timer:      sudo systemctl list-timers roxagent.timer"
}

usage() {
    cat >&2 <<EOF
Usage:
  $0                                                # Install locally
  $0 [virtctl-flags...] <user@vmi/name>            # Remote install via virtctl (default)
  $0 --ssh <user@host> [port]                      # Remote install via SSH

Examples:
  $0 -n openshift-cnv cloud-user@vmi/rhel10-1
  $0 --ssh root@192.168.1.10
  $0 --ssh root@192.168.1.10 2222
EOF
}

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
    if [ $# -eq 0 ]; then
        echo "error: virtctl mode requires at least a target (e.g. cloud-user@vmi/rhel10-1)" >&2
        echo "" >&2
        usage
        exit 1
    fi

    # Expects virtctl flags + target, e.g.:
    #   -n openshift-cnv cloud-user@vmi/rhel10-1
    # The last positional arg is the target; everything else are flags.
    TRANSPORT_KIND="virtctl"
    VIRTCTL_TARGET="${*: -1}"

    local argc=$#
    if (( argc > 1 )); then
        # Collect all arguments except the last one (the target) as flags.
        VIRTCTL_FLAGS=("${@:1:argc-1}")
    else
        VIRTCTL_FLAGS=()
    fi
}

# --- Install logic (shared) --------------------------------------------------

create_remote_stage_dir() {
    local output
    output="$(
        remote_exec <<'SCRIPT'
set -euo pipefail
stage_dir="$(mktemp -d "${TMPDIR:-/var/tmp}/roxagent-install.XXXXXX")"
printf '__STAGE_DIR__=%s\n' "${stage_dir}"
SCRIPT
    )"

    local stage_dir
    stage_dir="$(printf '%s\n' "${output}" | sed -n 's/^__STAGE_DIR__=//p' | tail -n 1)"
    if [ -z "${stage_dir}" ]; then
        echo "failed to determine remote stage dir" >&2
        return 1
    fi
    printf '%s\n' "${stage_dir}"
}

copy_remote_stage_files() {
    local remote_stage_dir="$1"
    local file
    for file in "${STAGED_INSTALL_FILES[@]}"; do
        remote_copy "${SCRIPT_DIR}/${file}" "${remote_stage_dir}/${file}"
    done
}

validate_stage_dir() {
    local stage_dir="$1"
    local file
    for file in "${STAGED_INSTALL_FILES[@]}"; do
        [ "${file}" = "install.sh" ] && continue
        if [ ! -f "${stage_dir}/${file}" ]; then
            echo "missing staged file: ${stage_dir}/${file}" >&2
            return 1
        fi
    done
}

install_from_stage_dir() {
    local stage_dir="$1"
    validate_stage_dir "${stage_dir}"

    echo "Installing Quadlet units from ${stage_dir}..."

    # Quadlet container file (strip mounts for paths missing on this host)
    sudo mkdir -p /etc/containers/systemd/
    filter_container_file "${stage_dir}/roxagent.container" \
        | sudo tee /etc/containers/systemd/roxagent.container >/dev/null
    # restorecon resets SELinux labels so systemd/podman can read the new files.
    sudo restorecon -Rv /etc/containers/systemd/ 2>/dev/null || true

    # Timer and prep service go in standard systemd directory
    sudo cp "${stage_dir}/roxagent.timer" /etc/systemd/system/
    sudo cp "${stage_dir}/roxagent-prep.service" /etc/systemd/system/
    sudo restorecon -Rv /etc/systemd/system/roxagent.timer /etc/systemd/system/roxagent-prep.service 2>/dev/null || true

    # Recreate the lock directory on every boot since /run is tmpfs.
    sudo mkdir -p /etc/tmpfiles.d/
    sudo cp "${stage_dir}/roxagent-tmpfiles.conf" /etc/tmpfiles.d/roxagent.conf
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

install_local() {
    install_from_stage_dir "${SCRIPT_DIR}"
}

install_remote() {
    local remote_stage_dir

    echo "Preparing stage directory on target..."
    remote_stage_dir="$(create_remote_stage_dir)"

    echo "Copying files to target..."
    copy_remote_stage_files "${remote_stage_dir}"

    echo "Running install on target..."
    remote_exec <<SCRIPT
set -euo pipefail
cleanup() {
    rm -rf "${remote_stage_dir}"
}
trap cleanup EXIT
bash "${remote_stage_dir}/install.sh" --stage-dir "${remote_stage_dir}"
SCRIPT
}

# Produce a filtered roxagent.container on stdout, removing Volume= lines
# whose host source path does not exist on this machine.
filter_container_file() {
    local file="$1"
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

main "$@"
