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

# Verifies that at least one worker node advertises devices.kubevirt.io/kvm > 0.
# Must be called after CNV is installed (the device plugin registers the resource).
verify_kvm_available() {
    local kvm_capacity
    kvm_capacity="$(kubectl get nodes -o jsonpath='{.items[*].status.capacity.devices\.kubevirt\.io/kvm}' 2>/dev/null || true)"
    if [[ -z "$kvm_capacity" ]] || [[ "$kvm_capacity" =~ ^[[:space:]]*0*[[:space:]]*$ ]]; then
        die "No worker nodes with devices.kubevirt.io/kvm > 0. Nested virtualization may not be enabled on this cluster."
    fi
    info "KVM device capacity on nodes: ${kvm_capacity}"
}

# Downloads virtctl from the cluster's ConsoleCLIDownload if not already on PATH.
ensure_virtctl() {
    if command -v virtctl &>/dev/null; then
        info "virtctl already on PATH: $(command -v virtctl)"
        return
    fi
    local download_url
    download_url="$(kubectl get consoleclidownload virtctl-clidownloads-kubevirt-hyperconverged \
        -o jsonpath='{.spec.links[?(@.text=="Download virtctl for Linux for x86_64")].href}' 2>/dev/null || true)"
    if [[ -z "$download_url" ]]; then
        die "virtctl not found on PATH and ConsoleCLIDownload resource not available"
    fi
    info "Downloading virtctl from ${download_url}"
    local dest="/usr/local/bin"
    if [[ ! -w "$dest" ]]; then
        dest="$(mktemp -d)"
        export PATH="${dest}:${PATH}"
    fi
    curl -sSL "$download_url" | tar xz -C "$dest" virtctl
    chmod +x "${dest}/virtctl"
    info "virtctl installed at ${dest}/virtctl"
}

# Extracts roxagent from the main container image (already built by CI).
# Sets ROXAGENT_BINARY_PATH for the Go test suite.
ensure_roxagent() {
    if [[ -n "${ROXAGENT_BINARY_PATH:-}" ]] && [[ -x "${ROXAGENT_BINARY_PATH}" ]]; then
        info "roxagent already available at ${ROXAGENT_BINARY_PATH}"
        return
    fi

    local default_path="${_VM_SCANNING_LIB_ROOT}/bin/linux_amd64/roxagent"
    if [[ -x "$default_path" ]]; then
        info "roxagent found at default build path: ${default_path}"
        export ROXAGENT_BINARY_PATH="$default_path"
        return
    fi

    require_environment "MAIN_IMAGE"
    local roxagent_dir
    roxagent_dir="$(mktemp -d)"
    info "Extracting roxagent from ${MAIN_IMAGE}"
    oc image extract "${MAIN_IMAGE}" --path="/stackrox/bin/roxagent:${roxagent_dir}" --confirm
    chmod +x "${roxagent_dir}/roxagent"
    export ROXAGENT_BINARY_PATH="${roxagent_dir}/roxagent"
    info "roxagent extracted to ${ROXAGENT_BINARY_PATH}"
}
