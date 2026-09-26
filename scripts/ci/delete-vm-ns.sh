#!/usr/bin/env bash

set -euo pipefail

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"

usage() {
    cat <<EOF
$0 deletes VM-scanning e2e namespaces. Best-effort: exits 0
when kubectl cannot list namespaces or a delete fails.

Usage:
    $0
    $0 -h|--help

Environment:
    VM_SCAN_NAMESPACE_PREFIX  default: vm-scan-e2e
EOF
}

delete_vm_scan_namespaces() {
    local ns ns_list
    info ">>> Cleaning up VM scan e2e namespaces <<<"
    ns_list="$(mktemp)"
    if ! list_vm_scan_namespaces > "$ns_list"; then
        info "Skipping namespace delete: kubectl get ns failed"
        rm -f "$ns_list"
        return 0
    fi
    while IFS= read -r ns; do
        [[ -z "$ns" ]] && continue
        info "Deleting namespace ${ns}"
        kubectl delete namespace "$ns" --wait=false --request-timeout=60s </dev/null 2>&1 || \
            info "Namespace delete for ${ns} failed or already removed"
    done < "$ns_list"
    rm -f "$ns_list"
}

main() {
    case "${1:-}" in
        -h|--help)
            usage
            exit 0
            ;;
        "")
            ;;
        *)
            usage >&2
            exit 1
            ;;
    esac

    cd "$SCRIPTS_ROOT"
    delete_vm_scan_namespaces
    exit 0
}

main "$@"
