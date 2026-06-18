#!/usr/bin/env bash
# Main orchestrator for adding VMs to an ACS cluster and installing roxagent.
#
# Usage:
#   add-vms.sh [options]
#
# Options:
#   --cluster NAME       Infra cluster name (calls infractl for kubeconfig)
#   --num-vms N          Number of VMs (default: 1)
#   --os OS              VM OS: rhel9|rhel10 (default: rhel9)
#   --ssh-key PATH       Path to user SSH public key to add to target VMs
#   --namespace NS       Target namespace (default: openshift-cnv)
#   --vm-prefix PREFIX   VM name prefix (default: same as OS)
#   --artifacts-dir DIR  Path to infractl artifacts directory
#
# Environment:
#   KUBECONFIG                   If set and --cluster not given, uses this directly
#   STACKROX_REPO                Required for native agent build (default: repo root)
#   QUAY_RHACS_ENG_RO_USERNAME   Required; used to create VM image pull secret
#   QUAY_RHACS_ENG_RO_PASSWORD   Required; used to create VM image pull secret

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Defaults
CLUSTER_NAME=""
NUM_VMS=1
VM_OS="rhel9"
USER_SSH_KEY_PATH=""
NAMESPACE="openshift-cnv"
VM_PREFIX=""
ARTIFACTS_DIR=""
NATIVE_AGENT_READY_VMS=()
NATIVE_AGENT_FAILED_VMS=()

die() { echo "ERROR: $*" >&2; exit 1; }

# Prints the usage block from the file header comments and exits.
usage() {
    sed -n '2,/^$/s/^# //p' "${BASH_SOURCE[0]}" >&2
    exit 1
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --cluster)      CLUSTER_NAME="$2"; shift 2 ;;
            --num-vms)      NUM_VMS="$2"; shift 2 ;;
            --os)           VM_OS="$2"; shift 2 ;;
            --ssh-key)      USER_SSH_KEY_PATH="$2"; shift 2 ;;
            --namespace)    NAMESPACE="$2"; shift 2 ;;
            --vm-prefix)    VM_PREFIX="$2"; shift 2 ;;
            --artifacts-dir) ARTIFACTS_DIR="$2"; shift 2 ;;
            --help|-h)      usage ;;
            *)              die "Unknown option: $1" ;;
        esac
    done

    VM_PREFIX="${VM_PREFIX:-${VM_OS}}"

    # The workflow uses type: number, but gh CLI and the API still accept
    # arbitrary strings, so we validate here as well.
    if ! [[ "$NUM_VMS" =~ ^[0-9]+$ ]] || (( NUM_VMS < 1 )); then
        die "--num-vms must be a positive integer"
    fi
    [[ "$VM_OS" =~ ^rhel(9|10)$ ]] || die "--os must be rhel9 or rhel10"
}

# Sets KUBECONFIG: either fetches it from infractl for the given cluster
# name, or expects it to be pre-set in the environment.
# Validates that kubectl can reach the cluster before returning.
resolve_kubeconfig() {
    if [[ -n "$CLUSTER_NAME" ]]; then
        command -v infractl &>/dev/null || die "infractl required when --cluster is used"
        echo "Fetching artifacts for cluster '$CLUSTER_NAME'..."
        ARTIFACTS_DIR="${ARTIFACTS_DIR:-$(mktemp -d)}"
        infractl artifacts "$CLUSTER_NAME" --download-dir "$ARTIFACTS_DIR" >/dev/null
        export KUBECONFIG="${ARTIFACTS_DIR}/kubeconfig"
        echo "KUBECONFIG set to $KUBECONFIG"
    fi

    [[ -n "${KUBECONFIG:-}" ]] || die "KUBECONFIG not set. Use --cluster or set KUBECONFIG."
    kubectl cluster-info &>/dev/null || die "Cannot connect to cluster via KUBECONFIG=$KUBECONFIG"
}

resolve_user_ssh_key() {
    if [[ -n "$USER_SSH_KEY_PATH" ]]; then
        [[ -f "$USER_SSH_KEY_PATH" ]] || die "SSH public key file not found: $USER_SSH_KEY_PATH"
        USER_SSH_PUBLIC_KEY="$(cat "$USER_SSH_KEY_PATH")"
        export USER_SSH_PUBLIC_KEY
        echo "User SSH key loaded from $USER_SSH_KEY_PATH"
    elif [[ -n "${USER_SSH_PUBLIC_KEY:-}" ]]; then
        echo "User SSH key loaded from environment."
    fi
}

print_summary() {
    echo ""
    echo "==========================================="
    echo "  Add VMs to Cluster — Summary"
    echo "==========================================="
    echo ""
    echo "Namespace:    $NAMESPACE"
    echo "VM OS:        $VM_OS"
    echo "VM prefix:    $VM_PREFIX"
    echo "Agent type:   native"
    echo "Num VMs:      $NUM_VMS"
    echo ""

    if [[ ${#MANAGED_VMS[@]} -gt 0 ]]; then
        echo "Managed VMs (automation key access):"
        for vm in "${MANAGED_VMS[@]}"; do echo "  - $vm"; done
    fi
    if [[ ${#ADOPTED_VMS[@]} -gt 0 ]]; then
        echo "Adopted VMs (password fallback, key now injected):"
        for vm in "${ADOPTED_VMS[@]}"; do echo "  - $vm"; done
    fi
    if [[ ${#SKIPPED_VMS[@]} -gt 0 ]]; then
        echo "Skipped VMs (inaccessible):"
        for vm in "${SKIPPED_VMS[@]}"; do echo "  - $vm"; done
        echo ""
        echo "To grant access to skipped VMs, add the automation public key"
        echo "to ~/.ssh/authorized_keys on each VM."
    fi

    echo ""
    echo "--- SSH Access ---"
    echo ""
    echo "Fetch automation private key:"
    echo "  kubectl get secret ${AUTOMATION_SSH_SECRET:-acs-vm-automation-ssh} -n $NAMESPACE \\"
    echo "    -o jsonpath='{.data.id_ed25519}' | base64 -d > ./add-vms-id_ed25519"
    echo "  chmod 600 ./add-vms-id_ed25519"
    echo ""
    echo "SSH into a VM:"
    echo "  virtctl ssh -n $NAMESPACE --identity-file ./add-vms-id_ed25519 ${SSH_USER:-cloud-user}@vmi/${VM_PREFIX}-1"
    echo ""
    echo "Check VM status:"
    echo "  kubectl get vm,vmi -n $NAMESPACE"
    echo ""
    echo "==========================================="

    write_github_summary
}

# Appends a markdown heading + bullet list to GITHUB_STEP_SUMMARY.
# No-op when the list is empty. Args: heading, items...
append_summary_list() {
    local heading="$1"
    shift

    if [[ $# -eq 0 ]]; then
        return 0
    fi

    {
        echo "$heading"
        local item
        for item in "$@"; do
            echo "- \`$item\`"
        done
        echo ""
    } >> "$GITHUB_STEP_SUMMARY"
}

# Writes the GitHub Actions job summary (markdown) with run configuration,
# VM categories, agent health, and SSH access instructions.
# No-op when GITHUB_STEP_SUMMARY is unset (i.e. local runs).
write_github_summary() {
    if [[ -z "${GITHUB_STEP_SUMMARY:-}" ]]; then
        return 0
    fi

    {
        echo "## Add VMs to Cluster"
        echo ""
        echo "- Namespace: \`$NAMESPACE\`"
        echo "- VM OS: \`$VM_OS\`"
        echo "- VM prefix: \`$VM_PREFIX\`"
        echo "- Agent type: \`native\`"
        echo "- Requested VMs: \`$NUM_VMS\`"
        echo ""
    } >> "$GITHUB_STEP_SUMMARY"

    append_summary_list "### Managed VMs (automation key access)" "${MANAGED_VMS[@]}"
    append_summary_list "### Adopted VMs (password fallback, key now injected)" "${ADOPTED_VMS[@]}"
    append_summary_list "### Skipped VMs (inaccessible)" "${SKIPPED_VMS[@]}"

    {
        echo "### Native agent service verification"
        echo ""
    } >> "$GITHUB_STEP_SUMMARY"
    append_summary_list "Successfully started on:" "${NATIVE_AGENT_READY_VMS[@]}"
    append_summary_list "Needs attention:" "${NATIVE_AGENT_FAILED_VMS[@]}"

    {
        echo "### SSH access"
        echo ""
        echo "Download the automation private key:"
        echo ""
        echo '```bash'
        echo "kubectl get secret ${AUTOMATION_SSH_SECRET:-acs-vm-automation-ssh} -n $NAMESPACE \\"
        echo "  -o jsonpath='{.data.id_ed25519}' | base64 -d > ./add-vms-id_ed25519"
        echo "chmod 600 ./add-vms-id_ed25519"
        echo '```'
        echo ""
        echo "SSH into a VM:"
        echo ""
        echo '```bash'
        echo "virtctl ssh -n $NAMESPACE --identity-file ./add-vms-id_ed25519 ${SSH_USER:-cloud-user}@vmi/${VM_PREFIX}-1"
        echo '```'
        echo ""
        echo "Check VM status:"
        echo ""
        echo '```bash'
        echo "kubectl get vm,vmi -n $NAMESPACE"
        echo '```'
        echo ""
    } >> "$GITHUB_STEP_SUMMARY"

    echo "::notice title=VM action summary::See the job summary for VM access, key download, and native service verification."
}

cleanup() {
    rm -f "${AUTOMATION_SSH_PRIVKEY:-}" "${AUTOMATION_SSH_PUBKEY:-}"
}
trap cleanup EXIT

main() {
    echo "=== Add VMs to ACS Cluster ==="
    echo ""

    parse_args "$@"
    resolve_kubeconfig
    resolve_user_ssh_key

    echo ""
    echo "Configuration:"
    echo "  Namespace:   $NAMESPACE"
    echo "  VM OS:       $VM_OS"
    echo "  VM prefix:   $VM_PREFIX"
    echo "  Num VMs:     $NUM_VMS"
    echo "  Agent type:  native"
    echo ""

    # Export variables for sourced scripts
    export NAMESPACE VM_OS VM_PREFIX NUM_VMS ARTIFACTS_DIR
    export CONTAINER_IMAGE="quay.io/rhacs-eng/vm-images:${VM_OS}-dnf-primed-latest"

    # Step 1: Install virt operator (skippable when action handles this separately)
    if [[ "${SKIP_VIRT_OPERATOR:-false}" != "true" ]]; then
        # shellcheck source=install-virt-operator.sh
        source "$SCRIPT_DIR/install-virt-operator.sh"
        install_virt_operator
        echo ""
    else
        echo "Skipping virt operator install (handled externally)."
    fi

    # Step 2: Deploy VMs
    # shellcheck source=deploy-vms.sh
    source "$SCRIPT_DIR/deploy-vms.sh"
    deploy_vms
    echo ""

    # Step 3: Install agent
    local accessible_vms=()
    accessible_vms+=("${MANAGED_VMS[@]}" "${ADOPTED_VMS[@]}")

    if [[ ${#accessible_vms[@]} -eq 0 ]]; then
        die "No accessible VMs — cannot install agent. All VMs failed SSH readiness."
    else
        export AUTOMATION_SSH_PRIVKEY
        # shellcheck source=install-agent-native.sh
        source "$SCRIPT_DIR/install-agent-native.sh"
        install_agent_native "${accessible_vms[@]}"
    fi

    # Step 4: Summary
    print_summary
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
