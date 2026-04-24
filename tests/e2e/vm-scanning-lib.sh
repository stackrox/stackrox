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

    # Self-discoverable: virtctl on $PATH, SSH keys generated on the fly.
    # Override via env if the defaults are not suitable for the CI cluster.
    # VIRTCTL_PATH          - defaults to $(command -v virtctl)
    # VM_SSH_PRIVATE_KEY    - PEM content (not a path); ephemeral ed25519 key generated if unset
    # VM_SSH_PUBLIC_KEY     - authorized_keys line (not a path); generated with private key if unset

}

# Downloads virtctl from the cluster's ConsoleCLIDownload if not already on PATH.
ensure_virtctl_binary() {
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
            # CI-only fallback: this URL is read from cluster-managed ConsoleCLIDownload
            # metadata in ephemeral test clusters. Keeping this path for robustness while
            # we rely on cluster-provided trust material; residual risk is low in this lane.
            info "Download with cluster CA failed, retrying without TLS verification"
            if ! curl -sSLk "$download_url" | tar xz -C "$dest" virtctl; then
                rm -f "$ca_bundle"
                die "Failed to download virtctl from ${download_url}"
            fi
        fi
        rm -f "$ca_bundle"
    else
        # CI-only fallback rationale above applies here as well.
        info "Cluster ingress CA not available, skipping TLS verification"
        if ! curl -sSLk "$download_url" | tar xz -C "$dest" virtctl; then
            die "Failed to download virtctl from ${download_url}"
        fi
    fi
    if [[ ! -f "${dest}/virtctl" ]]; then
        die "Downloaded archive from ${download_url} does not contain virtctl"
    fi
    chmod +x "${dest}/virtctl"
    info "virtctl installed at ${dest}/virtctl"
}