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
    download_url="$(oc get consoleclidownload virtctl-clidownloads-kubevirt-hyperconverged \
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
    local ca_bundle
    ca_bundle="$(mktemp)"
    if oc get configmap -n openshift-config-managed default-ingress-cert \
            -o jsonpath='{.data.ca-bundle\.crt}' > "$ca_bundle" 2>/dev/null \
       && [[ -s "$ca_bundle" ]]; then
        info "Using ingress CA from default-ingress-cert configmap"
    elif oc get secret -n openshift-ingress-operator router-ca \
            -o jsonpath='{.data.tls\.crt}' 2>/dev/null | base64 -d > "$ca_bundle" \
         && [[ -s "$ca_bundle" ]]; then
        info "Using ingress CA from router-ca secret"
    else
        ca_bundle=""
    fi
    if [[ -n "$ca_bundle" ]]; then
        if ! curl -sSL --cacert "$ca_bundle" "$download_url" | tar xz -C "$dest" virtctl; then
            info "Download with cluster CA failed, retrying without TLS verification"
            curl -sSLk "$download_url" | tar xz -C "$dest" virtctl
        fi
        rm -f "$ca_bundle"
    else
        info "Cluster ingress CA not available, skipping TLS verification"
        curl -sSLk "$download_url" | tar xz -C "$dest" virtctl
    fi
    chmod +x "${dest}/virtctl"
    info "virtctl installed at ${dest}/virtctl"
}

# Ensures roxagent binary is available. Checks, in order:
# 1. ROXAGENT_BINARY_PATH already set and executable
# 2. Default build output at bin/linux_amd64/roxagent
# 3. Extract from the main container image (built by CI)
# 4. Build from source as last resort
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

    local main_image="${MAIN_IMAGE:-}"
    if [[ -z "$main_image" ]] && [[ -n "${MAIN_IMAGE_TAG:-}" ]]; then
        local repo="${MAIN_IMAGE_REPO:-$(make --quiet --no-print-directory -C "$_VM_SCANNING_LIB_ROOT" default-image-registry)/main}"
        main_image="${repo}:${MAIN_IMAGE_TAG}"
    fi

    if [[ -n "$main_image" ]]; then
        local roxagent_dir
        roxagent_dir="$(mktemp -d)"
        info "Extracting roxagent from ${main_image}"
        if oc image extract "${main_image}" --path="/stackrox/bin/roxagent:${roxagent_dir}" --confirm 2>/dev/null; then
            chmod +x "${roxagent_dir}/roxagent"
            export ROXAGENT_BINARY_PATH="${roxagent_dir}/roxagent"
            info "roxagent extracted to ${ROXAGENT_BINARY_PATH}"
            return
        fi
        info "oc image extract failed, falling back to build from source"
    fi

    info "Building roxagent from source"
    make -C "$_VM_SCANNING_LIB_ROOT" roxagent_linux-amd64
    export ROXAGENT_BINARY_PATH="$default_path"
    info "roxagent built at ${ROXAGENT_BINARY_PATH}"
}
