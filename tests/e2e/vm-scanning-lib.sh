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
    require_environment "VM_IMAGES"

    # Build a docker config JSON for pulling private container-disk images
    # (e.g. quay.io/rhacs-eng/vm-images/*) inside the VM test namespace.
    if [[ -n "${QUAY_RHACS_ENG_RO_USERNAME:-}" && -n "${QUAY_RHACS_ENG_RO_PASSWORD:-}" ]]; then
        local vm_pull_secret
        vm_pull_secret="$(mktemp)"
        cat > "$vm_pull_secret" <<EOF
{"auths":{"quay.io":{"username":"${QUAY_RHACS_ENG_RO_USERNAME}","password":"${QUAY_RHACS_ENG_RO_PASSWORD}"}}}
EOF
        export VM_IMAGE_PULL_SECRET_PATH="$vm_pull_secret"
        info "VM image pull secret written to ${vm_pull_secret}"
    else
        info "QUAY_RHACS_ENG_RO_USERNAME/PASSWORD not set; VM images must be publicly accessible"
    fi
}

_use_existing_virtctl_binary_if_available() {
    if [[ -n "${VIRTCTL_PATH:-}" ]]; then
        [[ -x "$VIRTCTL_PATH" ]] || die "VIRTCTL_PATH is not executable: ${VIRTCTL_PATH}"
        export PATH="$(dirname "$VIRTCTL_PATH"):${PATH}"
        info "Using virtctl from VIRTCTL_PATH: ${VIRTCTL_PATH}"
        return 0
    fi

    if command -v virtctl &>/dev/null; then
        info "virtctl already on PATH: $(command -v virtctl)"
        return 0
    fi

    return 1
}

# Downloads virtctl from ConsoleCLIDownload using verified TLS only.
# This function intentionally does not use curl -k fallback.
ensure_virtctl_binary() {
    _use_existing_virtctl_binary_if_available && return

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
        rm -f "$ca_bundle"
        die "Cluster ingress CA not available for verified virtctl download from ${download_url}"
    fi

    if ! curl -sSL --cacert "$ca_bundle" "$download_url" | tar xz -C "$dest" virtctl; then
        rm -f "$ca_bundle"
        die "Failed to download virtctl with verified TLS from ${download_url}"
    fi
    rm -f "$ca_bundle"
    if [[ ! -f "${dest}/virtctl" ]]; then
        die "Downloaded archive from ${download_url} does not contain virtctl"
    fi
    chmod +x "${dest}/virtctl"
    info "virtctl installed at ${dest}/virtctl"
}

# Downloads virtctl from ConsoleCLIDownload with curl -k.
# SECURITY RISK ACCEPTANCE:
# - TLS verification is intentionally disabled and this is vulnerable to MITM.
# - Used only as fallback when the verified helper fails in this VM-scanning lane.
# - Accepted here for ephemeral CI clusters where the URL comes from cluster-managed
#   ConsoleCLIDownload metadata but cluster trust material can still be unreliable.
# - Never use this helper for persistent/shared environments.
ensure_virtctl_binary_insecure() {
    _use_existing_virtctl_binary_if_available && return

    local download_url
    download_url="$(oc get consoleclidownload virtctl-clidownloads-kubevirt-hyperconverged \
        -o jsonpath='{.spec.links[?(@.text=="Download virtctl for Linux for x86_64")].href}' 2>/dev/null || true)"
    if [[ -z "$download_url" ]]; then
        die "virtctl not found on PATH and ConsoleCLIDownload resource not available"
    fi
    info "Downloading virtctl from ${download_url} with TLS verification disabled (insecure mode)"
    local dest="/usr/local/bin"
    if [[ ! -w "$dest" ]]; then
        dest="$(mktemp -d)"
        export PATH="${dest}:${PATH}"
    fi

    if ! curl -sSLk "$download_url" | tar xz -C "$dest" virtctl; then
        die "Failed to download virtctl from ${download_url} in insecure mode"
    fi
    if [[ ! -f "${dest}/virtctl" ]]; then
        die "Downloaded archive from ${download_url} does not contain virtctl"
    fi
    chmod +x "${dest}/virtctl"
    info "virtctl installed at ${dest}/virtctl (insecure download mode)"
}
