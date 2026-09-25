#!/usr/bin/env bash

set -euo pipefail

# collect-vm-guest-logs.sh copies roxagent journals from VM-scanning e2e guests
# via virtctl ssh. Best-effort: always exits 0.
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
    local pointed
    pointed="$(vm_scan_e2e_dir)/virtctl-path"
    if [[ -f "$pointed" ]]; then
        local src
        src="$(<"$pointed")"
        if [[ -n "$src" && -x "$src" ]]; then
            printf '%s\n' "$src"
            return 0
        fi
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

# list_vm_scan_vmis prints "namespace name" lines for VMIs in vm-scan-e2e namespaces.
# Returns 1 when kubectl cannot list VMIs (distinct from zero matches).
list_vm_scan_vmis() {
    local prefix="${VM_SCAN_NAMESPACE_PREFIX:-vm-scan-e2e}"
    local out err ns name
    out="$(mktemp)"
    err="$(mktemp)"
    if ! kubectl --request-timeout=30s get vmi -A \
        -o jsonpath='{range .items[*]}{.metadata.namespace}{" "}{.metadata.name}{"\n"}{end}' \
        > "$out" 2>"$err"; then
        info "kubectl get vmi failed: $(<"$err")" >&2
        rm -f "$out" "$err"
        return 1
    fi
    rm -f "$err"
    while read -r ns name; do
        [[ -z "$ns" || -z "$name" ]] && continue
        if [[ "$ns" == "$prefix" || "$ns" == "${prefix}-"* ]]; then
            printf '%s %s\n' "$ns" "$name"
        fi
    done < "$out"
    rm -f "$out"
    return 0
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

guest_ssh() {
    local virtctl_bin="$1"
    local identity="$2"
    local guest_user="$3"
    local ns="$4"
    local vmi="$5"
    local ssh_timeout="$6"
    local remote_cmd="$7"

    # virtctl ssh reads stdin (OpenSSH without -n) and would consume the VMI
    # list from the caller's while-read loop.
    run_with_timeout "$ssh_timeout" \
        "${virtctl_bin}" ssh \
        --namespace "$ns" \
        --identity-file "$identity" \
        --local-ssh-opts="-n" \
        --local-ssh-opts="-o StrictHostKeyChecking=no" \
        --local-ssh-opts="-o IdentitiesOnly=yes" \
        --local-ssh-opts="-o UserKnownHostsFile=/dev/null" \
        --local-ssh-opts="-o BatchMode=yes" \
        --local-ssh-opts="-o ConnectTimeout=30" \
        --username "$guest_user" \
        "vmi/${vmi}" \
        --command "$remote_cmd" \
        < /dev/null
}

# append_rhel8_container_journal adds container stdout that RHEL 8 Podman
# does not file on roxagent.service. Those entries are tagged systemd-roxagent.
# Lines already present in the unit journal are skipped.
append_rhel8_container_journal() {
    local virtctl_bin="$1"
    local identity="$2"
    local guest_user="$3"
    local ns="$4"
    local vmi="$5"
    local out_file="$6"
    local ssh_timeout="$7"
    local extra merged

    extra="$(mktemp)"
    merged="$(mktemp)"
    if guest_ssh "$virtctl_bin" "$identity" "$guest_user" "$ns" "$vmi" "$ssh_timeout" \
        "sudo journalctl -b --no-pager -o short-iso -t systemd-roxagent || true; sudo journalctl -b --no-pager -o short-iso CONTAINER_NAME=systemd-roxagent || true" \
        >"$extra" 2>/dev/null; then
        awk 'NR==FNR { seen[$0]=1; next } !seen[$0] && $0 !~ /^-- / { seen[$0]=1; print }' \
            "$out_file" "$extra" >"$merged"
        cat "$merged" >>"$out_file"
    fi
    rm -f "$extra" "$merged"
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

    if ! guest_ssh "$virtctl_bin" "$identity" "$guest_user" "$ns" "$vmi" "$ssh_timeout" \
        "sudo journalctl -u roxagent.service -u roxagent-serve.service -b --no-pager -o short-iso" \
        > "$out_file" 2>"$stderr_file"; then
        {
            echo "roxagent journal collection failed for ${ns}/${vmi}"
            echo "virtctl: ${virtctl_bin}"
            echo "guest_user: ${guest_user}"
            echo "--- stderr ---"
            cat "$stderr_file"
        } >> "$out_file"
        rm -f "$stderr_file"
        return 0
    fi
    rm -f "$stderr_file"
    append_rhel8_container_journal "$virtctl_bin" "$identity" "$guest_user" "$ns" "$vmi" "$out_file" "$ssh_timeout" || true
}

collect_journals() {
    local output_dir="$1"
    local guest_user="${VM_GUEST_USER:-cloud-user}"
    local virtctl_bin="" identity=""
    virtctl_bin="$(resolve_virtctl)" || virtctl_bin=""
    identity="$(resolve_ssh_identity)" || identity=""

    if [[ -n "$virtctl_bin" && -n "$identity" ]]; then
        info ">>> Collecting roxagent journals into ${output_dir} <<<"
        local vmi_count=0 ns vmi out_file vmi_list list_err
        vmi_list="$(mktemp)"
        list_err="$(mktemp)"
        if ! list_vm_scan_vmis > "$vmi_list" 2>"$list_err"; then
            info "Failed to list VMIs; writing collection-failed.txt"
            {
                echo "kubectl get vmi failed (not an empty match)"
                cat "$list_err"
            } > "${output_dir}/collection-failed.txt"
            rm -f "$vmi_list" "$list_err"
            return 0
        fi
        rm -f "$list_err"
        while read -r ns vmi; do
            [[ -z "$ns" || -z "$vmi" ]] && continue
            vmi_count=$((vmi_count + 1))
            out_file="${output_dir}/${ns}_${vmi}_roxagent.journal.log"
            info "Collecting roxagent journal for ${ns}/${vmi} -> ${out_file}"
            collect_roxagent_journal "$virtctl_bin" "$identity" "$guest_user" "$ns" "$vmi" "$out_file" || true
        done < "$vmi_list"
        rm -f "$vmi_list"

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
    exit 0
}

main "$@"
