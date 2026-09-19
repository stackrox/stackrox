#!/usr/bin/env bash
# Install roxagent on VMs using the Quadlet (container) method.
#
# Detects or accepts the ACS image tag, renders the correct Image= line
# in every image-bearing Quadlet file, and calls quadlet/install.sh
# for each target VM.
#
# Idempotent: reads the installed Image= tag on each VM and skips
# if it matches the desired tag.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
QUADLET_DIR="${SCRIPT_DIR}/quadlet"

# Globals (set by caller or defaulted)
NAMESPACE="${NAMESPACE:-openshift-cnv}"
SSH_USER="${SSH_USER:-cloud-user}"
IMAGE_TAG="${IMAGE_TAG:-}"
AUTOMATION_SSH_PRIVKEY="${AUTOMATION_SSH_PRIVKEY:-}"

# Image-bearing Quadlet files whose Image= line must be rendered.
# TODO: When the reactive agent lands, add "roxagent-reactive.container" here
# and update installed_image_tag_matches() to check each file individually.
IMAGE_BEARING_FILES=("roxagent.container")

# Set by render_quadlet_image_tag; points to temp copy with rendered Image= lines
RENDERED_QUADLET_DIR=""

die() { echo "ERROR: $*" >&2; exit 1; }

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

    # Extract tag from image reference (quay.io/stackrox-io/main:TAG or similar)
    IMAGE_TAG="${central_image##*:}"
    if [[ -z "$IMAGE_TAG" || "$IMAGE_TAG" == "$central_image" ]]; then
        die "Cannot parse tag from Central image: $central_image"
    fi
    echo "Detected image tag: $IMAGE_TAG"
}

render_quadlet_image_tag() {
    local full_image="quay.io/stackrox-io/main:${IMAGE_TAG}"
    echo "Rendering Image=${full_image} in Quadlet files..."

    # Work on a temp copy to avoid dirtying the git working tree
    RENDERED_QUADLET_DIR="$(mktemp -d)"
    cp -a "${QUADLET_DIR}/." "${RENDERED_QUADLET_DIR}/"

    for f in "${IMAGE_BEARING_FILES[@]}"; do
        local file="${RENDERED_QUADLET_DIR}/${f}"
        if [[ ! -f "$file" ]]; then
            echo "  WARNING: ${f} not found in ${QUADLET_DIR} — skipping."
            continue
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
        test -f /etc/systemd/system/roxagent.timer &&
        test -f /etc/systemd/system/roxagent-prep.service &&
        test -f /etc/tmpfiles.d/roxagent.conf &&
        systemctl is-enabled roxagent.timer >/dev/null
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
        "grep -h '^Image=' /etc/containers/systemd/roxagent*.container 2>/dev/null || true" \
        2>/dev/null | tr -d '[:space:]')"

    local desired_line="Image=${desired_image}"
    desired_line="$(echo "$desired_line" | tr -d '[:space:]')"

    [[ "$installed_image" == "$desired_line" ]]
}

install_on_vm() {
    local vm_name="$1"
    echo "--- Installing roxagent (Quadlet) on $vm_name ---"

    if installed_image_tag_matches "$vm_name"; then
        echo "  Image tag already matches on $vm_name — skipping."
        return 0
    fi

    QUADLET_FILES_DIR="$RENDERED_QUADLET_DIR" \
        "${RENDERED_QUADLET_DIR}/install.sh" --virtctl \
        -n "$NAMESPACE" \
        --identity-file "$AUTOMATION_SSH_PRIVKEY" \
        "${SSH_USER}@vmi/${vm_name}"

    echo "  Installed on $vm_name."
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
