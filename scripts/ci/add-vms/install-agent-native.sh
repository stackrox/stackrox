#!/usr/bin/env bash
# Install roxagent on VMs using the native binary method.
#
# Builds the roxagent binary from source and deploys via virtctl scp.
# Always overwrites (binary has no version info; binary is small).
#
# Requires: Go toolchain, STACKROX_REPO (defaults to repo root).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STACKROX_REPO="${STACKROX_REPO:-$(cd "$SCRIPT_DIR/../../.." && pwd)}"
ROXAGENT_SRC="${STACKROX_REPO}/compliance/virtualmachines/roxagent"

NAMESPACE="${NAMESPACE:-openshift-cnv}"
SSH_USER="${SSH_USER:-cloud-user}"
AUTOMATION_SSH_PRIVKEY="${AUTOMATION_SSH_PRIVKEY:-}"
NATIVE_AGENT_READY_VMS=()
NATIVE_AGENT_FAILED_VMS=()

NATIVE_MOUNT_CANDIDATES=(
    /etc/os-release
    /etc/redhat-release
    /etc/system-release-cpe
    /etc/pki/entitlement
    /etc/yum.repos.d
    /etc/yum/repos.d
    /etc/distro.repos.d
    /var/cache/dnf
    /var/lib/dnf
)

die() { echo "ERROR: $*" >&2; exit 1; }

build_ssh_opts() {
    _ssh_opts=(
        --namespace "$NAMESPACE"
        --identity-file "$AUTOMATION_SSH_PRIVKEY"
        --local-ssh-opts="-o StrictHostKeyChecking=no"
        --local-ssh-opts="-o UserKnownHostsFile=/dev/null"
        --local-ssh-opts="-o ConnectTimeout=10"
    )
}

build_agent() {
    echo "=== Building roxagent binary ===" >&2

    if [[ ! -d "$ROXAGENT_SRC" ]]; then
        die "roxagent source not found at $ROXAGENT_SRC — set STACKROX_REPO"
    fi

    command -v go &>/dev/null || die "Go toolchain required for native agent build"

    local output="/tmp/roxagent-amd64"
    echo "Building from ${ROXAGENT_SRC}..." >&2
    GOOS=linux GOARCH=amd64 go build -o "$output" "$ROXAGENT_SRC"
    echo "Built: $output" >&2
    echo "$output"
}

create_native_prep_service_file() {
    cat <<'EOF'
[Unit]
Description=Prepare native StackRox VM Agent inputs

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStartPre=/bin/rm -rf /tmp/roxroot
ExecStartPre=/bin/mkdir -p /tmp/roxroot/etc/pki /tmp/roxroot/etc/pki/entitlement
ExecStartPre=/bin/mkdir -p /tmp/roxroot/etc/yum.repos.d /tmp/roxroot/etc/yum/repos.d /tmp/roxroot/etc/distro.repos.d
ExecStartPre=/bin/mkdir -p /tmp/roxroot/var/lib /tmp/roxroot/var/lib/dnf
ExecStartPre=/bin/mkdir -p /tmp/roxroot/var/cache /tmp/roxroot/var/cache/dnf
ExecStartPre=/bin/mkdir -p /run/lock/roxagent
ExecStartPre=/bin/touch /tmp/roxroot/etc/os-release /tmp/roxroot/etc/redhat-release /tmp/roxroot/etc/system-release-cpe

ExecStart=/bin/rm -rf /tmp/roxagent-rpm
ExecStart=/bin/cp -a /var/lib/rpm /tmp/roxagent-rpm
ExecStart=/bin/chmod -R 755 /tmp/roxagent-rpm
EOF
}

create_native_service_file() {
    local mount_path

    cat <<'EOF'
[Unit]
Description=StackRox VM Agent (native)
After=network.target roxagent-prep.service
Requires=roxagent-prep.service

[Service]
Type=oneshot
User=root
BindPaths=/tmp/roxagent-rpm:/tmp/roxroot/var/lib/rpm
EOF

    for mount_path in "$@"; do
        printf 'BindReadOnlyPaths=%s:/tmp/roxroot%s\n' "$mount_path" "$mount_path"
    done

    cat <<'EOF'
ExecStart=/usr/local/bin/roxagent --host-path /tmp/roxroot
StandardOutput=journal
StandardError=journal
EOF
}

create_native_timer_file() {
    cat <<'EOF'
[Unit]
Description=Run StackRox VM Agent periodically

[Timer]
OnBootSec=5min
OnUnitActiveSec=3h40m
RandomizedDelaySec=40min
Persistent=true

[Install]
WantedBy=timers.target
EOF
}

native_agent_service_verified() {
    local vm_name="$1"
    build_ssh_opts

    local status_output
    status_output="$(virtctl ssh "${_ssh_opts[@]}" \
        --command "service_result=\"\$(systemctl show roxagent.service -p Result --value 2>/dev/null || true)\";
timer_enabled=\"\$(systemctl is-enabled roxagent.timer 2>/dev/null || true)\";
timer_active=\"\$(systemctl is-active roxagent.timer 2>/dev/null || true)\";
printf \"%s\\n%s\\n%s\\n\" \"\$service_result\" \"\$timer_enabled\" \"\$timer_active\"" \
        "${SSH_USER}@vmi/${vm_name}" 2>/dev/null || true)"

    mapfile -t status_lines <<< "$status_output"

    [[ "${status_lines[0]:-}" == "success" &&
        "${status_lines[1]:-}" == "enabled" &&
        "${status_lines[2]:-}" == "active" ]]
}

install_on_vm() {
    local vm_name="$1" binary_path="$2"

    echo "--- Installing roxagent (native) on $vm_name ---"

    build_ssh_opts

    echo "  Probing host inputs for curated roxroot mount set..."
    local probe_cmd mount_path
    probe_cmd="for path in"
    for mount_path in "${NATIVE_MOUNT_CANDIDATES[@]}"; do
        probe_cmd+=" ${mount_path}"
    done
    probe_cmd+="; do [ -e \"\$path\" ] && printf \"%s\\n\" \"\$path\"; done"

    local -a available_mounts=()
    while IFS= read -r mount_path; do
        [[ -n "$mount_path" ]] && available_mounts+=("$mount_path")
    done < <(
        virtctl ssh "${_ssh_opts[@]}" \
            --command "$probe_cmd" \
            "${SSH_USER}@vmi/${vm_name}" 2>/dev/null || true
    )

    echo "  Copying binary..."
    virtctl scp "${_ssh_opts[@]}" \
        "$binary_path" \
        "${SSH_USER}@vmi/${vm_name}:/tmp/roxagent"

    echo "  Installing systemd units..."
    local service_file prep_service_file timer_file
    service_file="$(mktemp)"
    prep_service_file="$(mktemp)"
    timer_file="$(mktemp)"
    create_native_prep_service_file > "$prep_service_file"
    create_native_service_file "${available_mounts[@]}" > "$service_file"
    create_native_timer_file > "$timer_file"

    virtctl scp "${_ssh_opts[@]}" \
        "$prep_service_file" \
        "${SSH_USER}@vmi/${vm_name}:/tmp/roxagent-prep.service"
    virtctl scp "${_ssh_opts[@]}" \
        "$service_file" \
        "${SSH_USER}@vmi/${vm_name}:/tmp/roxagent.service"
    virtctl scp "${_ssh_opts[@]}" \
        "$timer_file" \
        "${SSH_USER}@vmi/${vm_name}:/tmp/roxagent.timer"
    rm -f "$prep_service_file" "$service_file" "$timer_file"

    virtctl ssh "${_ssh_opts[@]}" \
        --command 'set -e
sudo install -m 0755 /tmp/roxagent /usr/local/bin/roxagent
sudo restorecon -v /usr/local/bin/roxagent 2>/dev/null || true
rm -f /tmp/roxagent
sudo cp /tmp/roxagent-prep.service /etc/systemd/system/roxagent-prep.service
sudo cp /tmp/roxagent.service /etc/systemd/system/roxagent.service
sudo cp /tmp/roxagent.timer /etc/systemd/system/roxagent.timer
sudo restorecon -Rv /etc/systemd/system/roxagent-prep.service /etc/systemd/system/roxagent.service /etc/systemd/system/roxagent.timer 2>/dev/null || true
sudo systemctl daemon-reload
sudo systemctl enable --now roxagent.timer
echo "NATIVE_INSTALL_OK"' \
        "${SSH_USER}@vmi/${vm_name}"

    echo "  Installed on $vm_name."
    echo "  Verifying agent status on $vm_name..."
    local verify_output
    verify_output="$(virtctl ssh "${_ssh_opts[@]}" \
        --command 'sudo systemctl start roxagent.service; echo "---"; sudo systemctl status roxagent.timer --no-pager; echo "---"; sudo journalctl -u roxagent.service --no-pager -n 20' \
        "${SSH_USER}@vmi/${vm_name}" 2>&1 || true)"
    printf '%s\n' "$verify_output"

    if native_agent_service_verified "$vm_name"; then
        NATIVE_AGENT_READY_VMS+=("$vm_name")
    else
        NATIVE_AGENT_FAILED_VMS+=("$vm_name")
    fi
}

install_agent_native() {
    local vm_names=("$@")

    local binary_path
    binary_path="$(build_agent)"

    for vm in "${vm_names[@]}"; do
        install_on_vm "$vm" "$binary_path"
    done

    rm -f "$binary_path"
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        echo "Usage: $0 <vm-name> [vm-name...]" >&2
        exit 1
    fi
    install_agent_native "$@"
fi
