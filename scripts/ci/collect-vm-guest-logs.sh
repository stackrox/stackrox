#!/usr/bin/env bash

set -euo pipefail

# collect-vm-guest-logs.sh copies roxagent journals from VM-scanning e2e guests
# via virtctl ssh, then deletes the test namespaces. Best-effort: always exits 0.
#
# Usage:
#   collect-vm-guest-logs.sh <output-dir>
#
# Environment:
#   VM_SCAN_NAMESPACE_PREFIX  default: vm-scan-e2e
#   VM_GUEST_USER             default: cloud-user
#   VM_SCAN_E2E_DIR           host files shared with the test run (default: /tmp/vm-scan-e2e)
#   VM_SSH_PRIVATE_KEY_PATH   PEM identity for virtctl (default: $VM_SCAN_E2E_DIR/ssh-identity)
#   VIRTCTL_PATH              optional virtctl override
#   VM_GUEST_JOURNAL_TIMEOUT  per-VM virtctl ssh ceiling (default: 90s)

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

usage() {
    echo "./scripts/ci/collect-vm-guest-logs.sh <output-dir>"
}

vm_scan_e2e_dir() {
    printf '%s\n' "${VM_SCAN_E2E_DIR:-/tmp/vm-scan-e2e}"
}

resolve_virtctl() {
    if [[ -n "${VIRTCTL_PATH:-}" ]]; then
        if [[ -x "$VIRTCTL_PATH" ]]; then
            printf '%s\n' "$VIRTCTL_PATH"
            return 0
        fi
        info "VIRTCTL_PATH is not executable: ${VIRTCTL_PATH}" >&2
        return 1
    fi
    local persisted
    persisted="$(vm_scan_e2e_dir)/virtctl"
    if [[ -x "$persisted" ]]; then
        printf '%s\n' "$persisted"
        return 0
    fi
    command -v virtctl
}

resolve_ssh_identity() {
    if [[ -n "${VM_SSH_PRIVATE_KEY_PATH:-}" && -f "${VM_SSH_PRIVATE_KEY_PATH}" ]]; then
        printf '%s\n' "${VM_SSH_PRIVATE_KEY_PATH}"
        return 0
    fi
    local persisted
    persisted="$(vm_scan_e2e_dir)/ssh-identity"
    if [[ -f "$persisted" ]]; then
        printf '%s\n' "$persisted"
        return 0
    fi
    return 1
}

list_vm_scan_namespaces() {
    local prefix="${VM_SCAN_NAMESPACE_PREFIX:-vm-scan-e2e}"
    kubectl --request-timeout=30s get ns -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' \
        | grep -E "^${prefix}(-|$)" || true
}

# list_vm_scan_vmis prints "namespace name" lines for VMIs in vm-scan-e2e namespaces.
list_vm_scan_vmis() {
    local prefix="${VM_SCAN_NAMESPACE_PREFIX:-vm-scan-e2e}"
    local ns name
    while read -r ns name; do
        [[ -z "$ns" || -z "$name" ]] && continue
        if [[ "$ns" == "$prefix" || "$ns" == "${prefix}-"* ]]; then
            printf '%s %s\n' "$ns" "$name"
        fi
    done < <(kubectl --request-timeout=30s get vmi -A \
        -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{"\n"}{end}' \
        2>/dev/null || true)
}

run_with_timeout() {
    local seconds="$1"
    shift
    if command -v timeout >/dev/null; then
        timeout -k 5s "${seconds}s" "$@"
    else
        "$@"
    fi
}

collect_roxagent_journal() {
    local virtctl_bin="$1"
    local identity="$2"
    local guest_user="$3"
    local ns="$4"
    local vmi="$5"
    local out_file="$6"
    local ssh_timeout="${VM_GUEST_JOURNAL_TIMEOUT:-90}"

    local stderr_file
    stderr_file="$(mktemp)"

    if ! run_with_timeout "$ssh_timeout" \
        "${virtctl_bin}" ssh \
        --namespace "$ns" \
        --identity-file "$identity" \
        --local-ssh-opts="-o StrictHostKeyChecking=no" \
        --local-ssh-opts="-o IdentitiesOnly=yes" \
        --local-ssh-opts="-o UserKnownHostsFile=/dev/null" \
        --local-ssh-opts="-o BatchMode=yes" \
        --local-ssh-opts="-o ConnectTimeout=30" \
        --username "$guest_user" \
        "vmi/${vmi}" \
        --command "sudo journalctl -u roxagent.service -b --no-pager -o short-iso" \
        > "$out_file" 2>"$stderr_file"; then
        {
            echo "roxagent journal collection failed for ${ns}/${vmi}"
            echo "virtctl: ${virtctl_bin}"
            echo "guest_user: ${guest_user}"
            echo "--- stderr ---"
            cat "$stderr_file"
        } > "$out_file"
    fi
    rm -f "$stderr_file"
}

delete_vm_scan_namespaces() {
    local ns
    info ">>> Cleaning up VM scan e2e namespaces <<<"
    while IFS= read -r ns; do
        [[ -z "$ns" ]] && continue
        info "Deleting namespace ${ns}"
        kubectl delete namespace "$ns" --wait=false --request-timeout=60s 2>&1 || \
            info "Namespace delete for ${ns} failed or already removed"
    done < <(list_vm_scan_namespaces)
}

collect_journals() {
    local output_dir="$1"
    local guest_user="${VM_GUEST_USER:-cloud-user}"
    local virtctl_bin="" identity=""
    virtctl_bin="$(resolve_virtctl)" || virtctl_bin=""
    identity="$(resolve_ssh_identity)" || identity=""

    if [[ -n "$virtctl_bin" && -n "$identity" ]]; then
        info ">>> Collecting roxagent journals into ${output_dir} <<<"
        local vmi_count=0 ns vmi out_file
        while read -r ns vmi; do
            [[ -z "$ns" || -z "$vmi" ]] && continue
            vmi_count=$((vmi_count + 1))
            out_file="${output_dir}/${ns}_${vmi}_roxagent.journal.log"
            info "Collecting roxagent journal for ${ns}/${vmi} -> ${out_file}"
            collect_roxagent_journal "$virtctl_bin" "$identity" "$guest_user" "$ns" "$vmi" "$out_file" || true
        done < <(list_vm_scan_vmis)

        if [[ "$vmi_count" -eq 0 ]]; then
            info "No VMIs found in VM scan namespaces"
            echo "no VMIs matched prefix ${VM_SCAN_NAMESPACE_PREFIX:-vm-scan-e2e}" \
                > "${output_dir}/collection-skipped.txt"
        fi
        return 0
    fi

    if [[ -z "$virtctl_bin" ]]; then
        info "virtctl not found; skipping VM guest journal collection"
        echo "virtctl not found" > "${output_dir}/collection-skipped.txt"
    else
        info "VM SSH identity not found; skipping VM guest journal collection"
        echo "ssh identity not found at $(vm_scan_e2e_dir)/ssh-identity" \
            > "${output_dir}/collection-skipped.txt"
    fi
}

main() {
    cd "$SCRIPTS_ROOT"

    if [[ $# -lt 1 ]]; then
        usage
        exit 1
    fi

    local output_dir="$1"
    mkdir -p "$output_dir"
    collect_journals "$output_dir" || true
    delete_vm_scan_namespaces
    exit 0
}

main "$@"
