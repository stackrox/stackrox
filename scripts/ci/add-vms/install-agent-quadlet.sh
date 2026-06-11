#!/usr/bin/env bash
# Install roxagent on VMs using the Quadlet (container) method.
#
# Renders the cluster main image tag into roxagent.container and runs the
# customer install script from compliance/virtualmachines/roxagent/quadlet.
#
# Idempotent: reads the installed Image= tag on each VM and skips when it
# matches and roxagent.service is already active.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STACKROX_REPO="${STACKROX_REPO:-$(cd "$SCRIPT_DIR/../../.." && pwd)}"
QUADLET_DIR="${STACKROX_REPO}/compliance/virtualmachines/roxagent/quadlet"

# Globals (set by caller or defaulted)
NAMESPACE="${NAMESPACE:-openshift-cnv}"
SSH_USER="${SSH_USER:-cloud-user}"
IMAGE_TAG="${IMAGE_TAG:-}"
AUTOMATION_SSH_PRIVKEY="${AUTOMATION_SSH_PRIVKEY:-}"
QUADLET_AGENT_READY_VMS=()
QUADLET_AGENT_FAILED_VMS=()

IMAGE_BEARING_FILES=("roxagent.container")

# Set by render_quadlet_image_tag; points to temp copy with rendered Image= lines
RENDERED_QUADLET_DIR=""

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

virtctl_ssh() {
    local vm_name="$1"
    local remote_cmd="$2"

    virtctl ssh \
        --namespace "$NAMESPACE" \
        --identity-file "$AUTOMATION_SSH_PRIVKEY" \
        --local-ssh-opts="-o StrictHostKeyChecking=no" \
        --local-ssh-opts="-o UserKnownHostsFile=/dev/null" \
        --local-ssh-opts="-o BatchMode=yes" \
        --local-ssh-opts="-o ConnectTimeout=10" \
        --command "$remote_cmd" \
        "${SSH_USER}@vmi/${vm_name}"
}

cleanup_rendered_quadlet_dir() {
    rm -rf "${RENDERED_QUADLET_DIR:-}"
}

detect_image_tag() {
    if [[ -n "$IMAGE_TAG" ]]; then
        echo "Using provided image tag: $IMAGE_TAG"
        return 0
    fi

    echo "Auto-detecting image tag from Central deployment..."
    local central_image
    central_image="$(kubectl get deploy/central -n stackrox \
        -o jsonpath='{.spec.template.spec.containers[0].image}' 2>/dev/null || true)"

    if [[ -z "$central_image" ]]; then
        die "Cannot auto-detect image tag: Central deployment not found in stackrox namespace. Provide --image-tag explicitly."
    fi

    IMAGE_TAG="${central_image##*:}"
    if [[ -z "$IMAGE_TAG" || "$IMAGE_TAG" == "$central_image" ]]; then
        die "Cannot parse tag from Central image: $central_image"
    fi
    echo "Detected image tag: $IMAGE_TAG"
}

render_quadlet_image_tag() {
    local full_image="quay.io/stackrox-io/main:${IMAGE_TAG}"
    echo "Rendering Image=${full_image} in Quadlet files..."

    [[ -d "$QUADLET_DIR" ]] || die "Quadlet assets not found at $QUADLET_DIR — set STACKROX_REPO"

    RENDERED_QUADLET_DIR="$(mktemp -d)"
    cp -a "${QUADLET_DIR}/." "${RENDERED_QUADLET_DIR}/"

    for f in "${IMAGE_BEARING_FILES[@]}"; do
        local file="${RENDERED_QUADLET_DIR}/${f}"
        if [[ ! -f "$file" ]]; then
            die "Missing Quadlet file: ${QUADLET_DIR}/${f}"
        fi
        sed -i.bak "s|^Image=.*|Image=${full_image}|" "$file"
        rm -f "${file}.bak"
        echo "  Rendered: $f"
    done
}

quadlet_install_is_complete() {
    local vm_name="$1"
    local check_cmd="
        test -f /etc/containers/systemd/roxagent.container &&
        systemctl is-active roxagent.service >/dev/null
    "

    virtctl_ssh "$vm_name" "$check_cmd" >/dev/null 2>/dev/null
}

installed_image_tag_matches() {
    local vm_name="$1"
    local desired_image="quay.io/stackrox-io/main:${IMAGE_TAG}"

    if ! quadlet_install_is_complete "$vm_name"; then
        return 1
    fi

    local installed_image
    installed_image="$(virtctl_ssh "$vm_name" \
        "grep -h '^Image=' /etc/containers/systemd/roxagent.container 2>/dev/null || true" \
        2>/dev/null | tr -d '[:space:]')"

    local desired_line="Image=${desired_image}"
    desired_line="$(echo "$desired_line" | tr -d '[:space:]')"

    [[ "$installed_image" == "$desired_line" ]]
}

quadlet_agent_service_verified() {
    local vm_name="$1"
    build_ssh_opts

    local status_output
    status_output="$(virtctl ssh "${_ssh_opts[@]}" \
        --command "active=\"\$(systemctl is-active roxagent.service 2>/dev/null || true)\";
printf \"%s\\n\" \"\$active\"" \
        "${SSH_USER}@vmi/${vm_name}" 2>/dev/null || true)"

    [[ "$(echo "$status_output" | head -n1 | tr -d '[:space:]')" == "active" ]]
}

install_on_vm() {
    local vm_name="$1"
    echo "--- Installing roxagent (Quadlet) on $vm_name ---"

    if installed_image_tag_matches "$vm_name"; then
        echo "  Image tag already matches on $vm_name — verifying service..."
        if quadlet_agent_service_verified "$vm_name"; then
            QUADLET_AGENT_READY_VMS+=("$vm_name")
            return 0
        fi
    fi

    if ! QUADLET_FILES_DIR="$RENDERED_QUADLET_DIR" \
        "${RENDERED_QUADLET_DIR}/install.sh" --virtctl \
        -n "$NAMESPACE" \
        --identity-file "$AUTOMATION_SSH_PRIVKEY" \
        "${SSH_USER}@vmi/${vm_name}"; then
        echo "  Quadlet install failed on $vm_name" >&2
        QUADLET_AGENT_FAILED_VMS+=("$vm_name")
        return
    fi

    echo "  Installed on $vm_name."
    echo "  Verifying agent status on $vm_name..."
    local verify_output
    build_ssh_opts
    verify_output="$(virtctl ssh "${_ssh_opts[@]}" \
        --command 'echo "---"; sudo systemctl status roxagent.service --no-pager; echo "---"; sudo journalctl -u roxagent.service --no-pager -n 20' \
        "${SSH_USER}@vmi/${vm_name}" 2>&1 || true)"
    printf '%s\n' "$verify_output"

    local retries=24
    local delay_secs=10
    while ! quadlet_agent_service_verified "$vm_name"; do
        ((retries--))
        if ((retries == 0)); then
            QUADLET_AGENT_FAILED_VMS+=("$vm_name")
            return
        fi
        sleep "$delay_secs"
    done
    QUADLET_AGENT_READY_VMS+=("$vm_name")
}

install_agent_quadlet() {
    local vm_names=("$@")

    detect_image_tag
    render_quadlet_image_tag

    for vm in "${vm_names[@]}"; do
        install_on_vm "$vm"
    done
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        echo "Usage: $0 <vm-name> [vm-name...]" >&2
        exit 1
    fi
    trap cleanup_rendered_quadlet_dir EXIT
    install_agent_quadlet "$@"
fi
