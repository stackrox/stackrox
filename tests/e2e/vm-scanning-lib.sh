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

    # Quadlet pulls the cluster main image. Construct MAIN_IMAGE the same way
    # deploy.sh does when the caller only exported MAIN_IMAGE_TAG.
    if [[ -z "${ROXAGENT_IMAGE:-}" && -z "${MAIN_IMAGE:-}" ]]; then
        if [[ -z "${MAIN_IMAGE_TAG:-}" ]]; then
            MAIN_IMAGE_TAG="$(make --quiet --no-print-directory -C "$_VM_SCANNING_LIB_ROOT" tag)"
            export MAIN_IMAGE_TAG
        fi
        if [[ -z "${MAIN_IMAGE_REPO:-}" ]]; then
            MAIN_IMAGE_REPO="$(make --quiet --no-print-directory -C "$_VM_SCANNING_LIB_ROOT" default-image-registry)/main"
            export MAIN_IMAGE_REPO
        fi
        export MAIN_IMAGE="${MAIN_IMAGE_REPO}:${MAIN_IMAGE_TAG}"
        info "MAIN_IMAGE set to ${MAIN_IMAGE} for Quadlet roxagent"
    fi

    # Kubernetes imagePullSecret (username/password dockerconfig) and a separate
    # containers-auth file for guest Podman. RHEL 8 Podman requires the auth field.
    if [[ -n "${QUAY_RHACS_ENG_RO_USERNAME:-}" && -n "${QUAY_RHACS_ENG_RO_PASSWORD:-}" ]]; then
        local vm_pull_secret vm_podman_auth quay_auth
        vm_pull_secret="$(mktemp)"
        vm_podman_auth="$(mktemp)"
        cat > "$vm_pull_secret" <<EOF
{"auths":{"quay.io":{"username":"${QUAY_RHACS_ENG_RO_USERNAME}","password":"${QUAY_RHACS_ENG_RO_PASSWORD}"}}}
EOF
        quay_auth="$(printf '%s:%s' "${QUAY_RHACS_ENG_RO_USERNAME}" "${QUAY_RHACS_ENG_RO_PASSWORD}" | base64 | tr -d '\n')"
        printf '{"auths":{"quay.io":{"auth":"%s"}}}\n' "$quay_auth" > "$vm_podman_auth"
        chmod 600 "$vm_podman_auth"
        export VM_IMAGE_PULL_SECRET_PATH="$vm_pull_secret"
        export VM_PODMAN_AUTH_FILE="$vm_podman_auth"
        info "VM image pull secret written to ${vm_pull_secret}"
        info "VM Podman auth file written to ${vm_podman_auth}"
    else
        info "QUAY_RHACS_ENG_RO_USERNAME/PASSWORD not set; VM images must be publicly accessible"
    fi
}

# Priority: explicit VIRTCTL_PATH override > implicit PATH discovery.
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

# Retrieves the cluster ingress CA bundle and prints its path to stdout.
# Any diagnostic logging must go to stderr so command substitution captures only the path.
# Dies if no trust material is available.
_fetch_cluster_ingress_ca() {
    local ca_bundle
    ca_bundle="$(mktemp)"
    if oc get configmap -n openshift-config-managed default-ingress-cert \
            -o jsonpath='{.data.ca-bundle\.crt}' > "$ca_bundle" 2>/dev/null \
       && [[ -s "$ca_bundle" ]]; then
        info "Using ingress CA from default-ingress-cert configmap" >&2
    elif oc get secret -n openshift-ingress-operator router-ca \
            -o jsonpath='{.data.tls\.crt}' 2>/dev/null | base64 -d > "$ca_bundle" \
         && [[ -s "$ca_bundle" ]]; then
        info "Using ingress CA from router-ca secret" >&2
    else
        rm -f "$ca_bundle"
        die "Cluster ingress CA not available"
    fi
    printf '%s\n' "$ca_bundle"
}

# Directory to install a downloaded virtctl binary into.
# PATH is not updated here: callers invoke this via command substitution, which
# would discard an export. Call _virtctl_add_install_dir_to_path after assigning dest.
_virtctl_install_dir() {
    local dest="/usr/local/bin"
    if [[ ! -w "$dest" ]]; then
        dest="$(mktemp -d)"
    fi
    printf '%s\n' "$dest"
}

_virtctl_add_install_dir_to_path() {
    local dest="$1"
    [[ "$dest" == "/usr/local/bin" ]] || export PATH="${dest}:${PATH}"
}

# Set to 1 when ConsoleCLIDownload returned a body that was not a gzip tarball.
# TLS retry (-k) cannot help in that case; callers should skip it.
_VIRTCTL_CLUSTER_DOWNLOAD_NONGZIP=0

# Downloads and installs virtctl using the provided curl TLS arguments.
# Returns 1 on failure (does not die) so callers can fall back.
# Usage: _download_and_install_virtctl [curl_tls_args...]
# Example: _download_and_install_virtctl --cacert /path/to/ca.pem
#          _download_and_install_virtctl -k
_download_and_install_virtctl() {
    # CI Prow workers are always Linux x86_64 (n2-standard-8 machine type).
    local dest
    dest="$(_virtctl_install_dir)"
    _virtctl_add_install_dir_to_path "$dest"
    _VIRTCTL_CLUSTER_DOWNLOAD_NONGZIP=0

    # ConsoleCLIDownload can exist while the ingress backend still serves HTML.
    # Wait for the href, then retry until the body is a gzip tarball.
    local download_url=""
    local attempt=0
    while (( attempt < 30 )); do
        attempt=$((attempt + 1))
        download_url="$(oc get consoleclidownload virtctl-clidownloads-kubevirt-hyperconverged \
            -o jsonpath='{.spec.links[?(@.text=="Download virtctl for Linux for x86_64")].href}' 2>/dev/null || true)"
        if [[ -n "$download_url" ]]; then
            break
        fi
        info "Waiting for ConsoleCLIDownload virtctl href (${attempt}/30)"
        sleep 10
    done
    if [[ -z "$download_url" ]]; then
        warn "virtctl not found on PATH and ConsoleCLIDownload resource not available"
        return 1
    fi

    local cli_deploy="hyperconverged-cluster-cli-download"
    if oc get deploy -n openshift-cnv "$cli_deploy" >/dev/null 2>&1; then
        info "Waiting for ${cli_deploy} rollout..."
        oc rollout status "deploy/${cli_deploy}" -n openshift-cnv --timeout=300s || true
    fi

    local archive saw_nongzip=0
    archive="$(mktemp)"
    info "Downloading virtctl from ${download_url}"
    attempt=0
    while (( attempt < 30 )); do
        attempt=$((attempt + 1))
        if curl -sSL --connect-timeout 30 --max-time 120 "$@" -o "$archive" "$download_url"; then
            if gzip -t "$archive" 2>/dev/null \
                && tar xz -C "$dest" virtctl -f "$archive" \
                && [[ -f "${dest}/virtctl" ]]; then
                rm -f "$archive"
                chmod +x "${dest}/virtctl"
                info "virtctl installed at ${dest}/virtctl"
                return 0
            fi
            saw_nongzip=1
        fi
        info "virtctl tarball not ready yet (attempt ${attempt}/30), retrying in 10s"
        sleep 10
    done
    rm -f "$archive"
    _VIRTCTL_CLUSTER_DOWNLOAD_NONGZIP=$saw_nongzip
    warn "Failed to download virtctl from ${download_url}"
    return 1
}

# True when $1 is a kubevirt/kubevirt GitHub release tag (v1.6.0 or 1.6.0).
# Image digests (sha256:...) and two-part versions (v1.6) are not release tags.
_is_kubevirt_github_release_tag() {
    [[ "$1" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]]
}

_normalize_kubevirt_github_tag() {
    local v="$1"
    if [[ "$v" == v* ]]; then
        printf '%s\n' "$v"
    else
        printf 'v%s\n' "$v"
    fi
}

# Prints a kubevirt/kubevirt GitHub release tag to stdout (logs go to stderr).
# CNV digest-pins operand images, so observedKubeVirtVersion is often sha256:...;
# those values are skipped. If the CR has no tag, use GitHub's latest release.
_kubevirt_github_release_tag() {
    local v
    while IFS= read -r v; do
        [[ -n "$v" ]] || continue
        if _is_kubevirt_github_release_tag "$v"; then
            _normalize_kubevirt_github_tag "$v"
            return 0
        fi
        warn "Ignoring KubeVirt version ${v}: not a GitHub release tag" >&2
    done <<EOF
$(oc get kubevirt -n openshift-cnv -o jsonpath='{.items[0].status.observedKubeVirtVersion}' 2>/dev/null || true)
$(oc get kubevirt -n openshift-cnv -o jsonpath='{.items[0].status.targetKubeVirtVersion}' 2>/dev/null || true)
$(oc get deploy virt-operator -n openshift-cnv -o jsonpath='{range .spec.template.spec.containers[*].env[?(@.name=="KUBEVIRT_VERSION")]}{.value}{"\n"}{end}' 2>/dev/null || true)
EOF

    local latest_url tag
    latest_url="$(curl -fsSL --connect-timeout 30 --max-time 60 -o /dev/null -w '%{url_effective}' \
        https://github.com/kubevirt/kubevirt/releases/latest)" || return 1
    tag="${latest_url##*/}"
    if ! _is_kubevirt_github_release_tag "$tag"; then
        warn "GitHub latest release URL did not end in a tag: ${latest_url}" >&2
        return 1
    fi
    tag="$(_normalize_kubevirt_github_tag "$tag")"
    info "No cluster KubeVirt GitHub tag; using latest release ${tag}" >&2
    printf '%s\n' "$tag"
    return 0
}

# Installs virtctl from a kubevirt/kubevirt GitHub release.
# ConsoleCLIDownload can stay on HTML while CNV's CSV is Installing or Failed.
_install_virtctl_from_kubevirt_release() {
    local version dest url bin
    version="$(_kubevirt_github_release_tag)" || {
        warn "Could not resolve a kubevirt/kubevirt GitHub release tag for virtctl"
        return 1
    }

    dest="$(_virtctl_install_dir)"
    _virtctl_add_install_dir_to_path "$dest"
    url="https://github.com/kubevirt/kubevirt/releases/download/${version}/virtctl-${version}-linux-amd64"
    bin="${dest}/virtctl"
    info "Downloading virtctl ${version} from ${url}"
    if ! curl -fsSL --connect-timeout 30 --max-time 120 --retry 5 --retry-delay 5 -o "$bin" "$url"; then
        warn "GitHub virtctl download failed from ${url}"
        rm -f "$bin"
        return 1
    fi
    if [[ "$(head -c 4 "$bin")" != $'\x7fELF' ]]; then
        warn "GitHub virtctl download was not an ELF binary"
        rm -f "$bin"
        return 1
    fi
    chmod +x "$bin"
    info "virtctl installed at ${bin}"
    return 0
}

# Downloads virtctl from ConsoleCLIDownload using verified TLS only.
ensure_virtctl_binary() {
    _use_existing_virtctl_binary_if_available && return

    local ca_bundle rc=0
    ca_bundle="$(_fetch_cluster_ingress_ca)"
    _download_and_install_virtctl --cacert "$ca_bundle" || rc=$?
    rm -f "$ca_bundle"
    return "$rc"
}

# Downloads virtctl from ConsoleCLIDownload with curl -k.
# SECURITY RISK ACCEPTANCE:
# - TLS verification is intentionally disabled and this is vulnerable to MITM.
# - Used only as fallback when the verified helper fails in this VM-scanning lane.
# - Accepted here for ephemeral CI clusters where the URL comes from cluster-managed
#   ConsoleCLIDownload metadata but cluster trust material can still be unreliable.
# - Never use this helper for persistent/shared environments; this is enforced by
#   rejecting the fallback outside CI.
ensure_virtctl_binary_insecure() {
    _use_existing_virtctl_binary_if_available && return
    is_CI || die "Secure virtctl download failed; refusing insecure curl -k fallback outside CI. Set VIRTCTL_PATH or fix cluster ingress trust material."
    _download_and_install_virtctl -k
}
