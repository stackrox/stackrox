#!/usr/bin/env bash
# VM scanning E2E cluster preflight helpers.
# shellcheck disable=SC1091

_VM_SCANNING_LIB_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/lib.sh
source "$_VM_SCANNING_LIB_ROOT/scripts/lib.sh"

# Ensures VM-scanning E2E required environment variables are set before deploy/tests.
# Variables with sensible defaults in the Go suite are optional here;
# only truly external inputs that cannot be self-discovered are required.
ensure_vm_scanning_cluster_prereqs() {
    require_environment "KUBECONFIG"
    require_environment "VM_IMAGE_RHEL9"
    require_environment "VM_IMAGE_RHEL10"

    # Self-discoverable: virtctl on $PATH, roxagent in build output, SSH keys generated on the fly.
    # Override via env if the defaults are not suitable for the CI cluster.
    # VIRTCTL_PATH          - defaults to $(command -v virtctl)
    # ROXAGENT_BINARY_PATH  - defaults to bin/linux_amd64/roxagent
    # VM_SSH_PRIVATE_KEY    - PEM content (not a path); ephemeral ed25519 key generated if unset
    # VM_SSH_PUBLIC_KEY     - authorized_keys line (not a path); generated with private key if unset

    # Activation: if both RHEL_ACTIVATION_ORG and RHEL_ACTIVATION_KEY are set,
    # the Go suite derives VM_SCAN_REQUIRE_ACTIVATION=true automatically.
    # Set VM_SCAN_REQUIRE_ACTIVATION explicitly only to force activation on/off.
}
