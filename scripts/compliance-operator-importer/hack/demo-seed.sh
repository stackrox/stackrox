#!/usr/bin/env bash
# demo-seed.sh — Seed demo fixtures for the CO → ACS importer.
#
# Creates 2 scan configs in ACS and 3 SSBs in Kubernetes. One SSB
# intentionally shares a name with an ACS scan config to demonstrate
# conflict handling.
#
# All resources are tagged with a short unique ID (e.g. "d7f2") so
# they can be identified and cleaned up reliably.
#
# Usage:
#   ./demo-seed.sh up        # create fixtures
#   ./demo-seed.sh down      # tear down fixtures
#   ./demo-seed.sh status    # show what exists
#
# Prerequisites:
#   ROX_ENDPOINT, ROX_ADMIN_PASSWORD (or ROX_API_TOKEN), kubectl access.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STATE_FILE="${SCRIPT_DIR}/.demo-seed-id"
CO_NS="openshift-compliance"

# ── ACS connection ───────────────────────────────────────────────────────────

ACS_ENDPOINT="${ROX_ENDPOINT:?ROX_ENDPOINT must be set}"
ACS_URL="${ACS_ENDPOINT#http://}"
ACS_URL="${ACS_URL#https://}"
ACS_URL="https://${ACS_URL}"

if [[ -n "${ROX_ADMIN_PASSWORD:-}" ]]; then
    CURL_AUTH=(-u "admin:${ROX_ADMIN_PASSWORD}")
elif [[ -n "${ROX_API_TOKEN:-}" ]]; then
    CURL_AUTH=(-H "Authorization: Bearer ${ROX_API_TOKEN}")
else
    echo "ERROR: set ROX_ADMIN_PASSWORD or ROX_API_TOKEN" >&2
    exit 1
fi

# ── Helpers ──────────────────────────────────────────────────────────────────

BOLD='\033[1m' DIM='\033[2m' GREEN='\033[32m' RED='\033[31m'
YELLOW='\033[33m' CYAN='\033[36m' RESET='\033[0m'

ok() { echo -e "  ${GREEN}✓${RESET} $1"; }
fail() { echo -e "  ${RED}✗${RESET} $1"; }
info() { echo -e "  ${DIM}$1${RESET}"; }
hdr() { echo -e "\n${CYAN}${BOLD}── $1 ──${RESET}\n"; }

acs_api() {
    local method="$1" path="$2"
    shift 2
    curl -sk "${CURL_AUTH[@]}" -X "$method" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        "${ACS_URL}${path}" "$@"
}

# Get the ACS cluster ID for the current context.
get_acs_cluster_id() {
    local clusters
    clusters=$(acs_api GET "/v1/clusters" 2> /dev/null)
    # Match by provider metadata (OpenShift cluster ID).
    local ocp_id
    ocp_id=$(kubectl get clusterversion version -o jsonpath='{.spec.clusterID}' 2> /dev/null || true)
    if [[ -n "$ocp_id" ]]; then
        local matched
        matched=$(echo "$clusters" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for c in data.get('clusters', []):
    pid = c.get('status',{}).get('providerMetadata',{}).get('cluster',{}).get('id','')
    if pid == '${ocp_id}':
        print(c['id']); break
" 2> /dev/null || true)
        if [[ -n "$matched" ]]; then
            echo "$matched"
            return
        fi
    fi
    # Fallback: first cluster.
    echo "$clusters" | python3 -c "
import sys, json
data = json.load(sys.stdin)
cs = data.get('clusters', [])
if cs: print(cs[0]['id'])
" 2> /dev/null
}

generate_id() {
    # Use od to avoid SIGPIPE from tr|head under pipefail.
    od -An -tx1 -N4 /dev/urandom | tr -d ' \n'
}

load_id() {
    if [[ ! -f "$STATE_FILE" ]]; then
        echo "ERROR: no active seed found (${STATE_FILE} missing). Run '$0 up' first." >&2
        exit 1
    fi
    cat "$STATE_FILE"
}

# Resource names derived from seed ID.
# ACS-only scans:        demo-{id}-stig-weekly, demo-{id}-cis-audit
# K8s SSBs:              demo-{id}-cis-audit (CONFLICT!), demo-{id}-moderate-daily, demo-{id}-pci-scan
# K8s ScanSetting:       demo-{id}-setting
names_for() {
    local id="$1"
    ACS_SCAN_1="demo-${id}-stig-weekly"
    ACS_SCAN_2="demo-${id}-cis-audit"
    SSB_1="demo-${id}-cis-audit" # same as ACS_SCAN_2 → conflict
    SSB_2="demo-${id}-moderate-daily"
    SSB_3="demo-${id}-pci-scan"
    SCAN_SETTING="demo-${id}-setting"
}

# ── UP ───────────────────────────────────────────────────────────────────────

cmd_up() {
    if [[ -f "$STATE_FILE" ]]; then
        local old_id
        old_id=$(cat "$STATE_FILE")
        echo -e "${YELLOW}WARNING: seed '${old_id}' already exists. Run '$0 down' first or '$0 up --force'.${RESET}"
        if [[ "${1:-}" != "--force" ]]; then exit 1; fi
        cmd_down
    fi

    local id
    id=$(generate_id)
    names_for "$id"

    echo -e "${BOLD}Seeding demo fixtures  [id: ${CYAN}${id}${RESET}${BOLD}]${RESET}"

    # ── K8s: ScanSetting ─────────────────────────────────────────────────
    hdr "Kubernetes: ScanSetting"
    kubectl apply -f - << EOF
apiVersion: compliance.openshift.io/v1alpha1
kind: ScanSetting
metadata:
  name: ${SCAN_SETTING}
  namespace: ${CO_NS}
  labels:
    demo-seed: "${id}"
schedule: "0 3 * * *"
roles: [worker, master]
rawResultStorage:
  rotation: 3
  size: 1Gi
EOF
    ok "${SCAN_SETTING}  (daily 03:00)"

    # ── K8s: SSBs ────────────────────────────────────────────────────────
    hdr "Kubernetes: ScanSettingBindings"
    for pair in \
        "${SSB_1}:ocp4-cis" \
        "${SSB_2}:ocp4-moderate" \
        "${SSB_3}:ocp4-pci-dss"; do
        local name="${pair%%:*}" profile="${pair#*:}"
        kubectl apply -f - << EOF
apiVersion: compliance.openshift.io/v1alpha1
kind: ScanSettingBinding
metadata:
  name: ${name}
  namespace: ${CO_NS}
  labels:
    demo-seed: "${id}"
profiles:
  - name: ${profile}
    kind: Profile
    apiGroup: compliance.openshift.io/v1alpha1
settingsRef:
  name: ${SCAN_SETTING}
  kind: ScanSetting
  apiGroup: compliance.openshift.io/v1alpha1
EOF
        local note=""
        [[ "$name" == "$SSB_1" ]] && note="  ← will conflict with ACS scan"
        ok "${name}  (${profile})${note}"
    done

    # ── ACS: scan configs ────────────────────────────────────────────────
    hdr "ACS: Scan Configurations"
    local cluster_id
    cluster_id=$(get_acs_cluster_id)
    if [[ -z "$cluster_id" ]]; then
        fail "Could not determine ACS cluster ID"
        exit 1
    fi
    info "Using ACS cluster ID: ${cluster_id}"

    # Scan 1: STIG weekly (no conflict with any SSB).
    acs_api POST "/v2/compliance/scan/configurations" -d "{
        \"scanName\": \"${ACS_SCAN_1}\",
        \"scanConfig\": {
            \"oneTimeScan\": false,
            \"profiles\": [\"ocp4-stig\"],
            \"scanSchedule\": {
                \"intervalType\": \"WEEKLY\",
                \"hour\": 4, \"minute\": 0,
                \"daysOfWeek\": { \"days\": [1] }
            },
            \"description\": \"Demo seed ${id}: STIG weekly scan (no conflict)\"
        },
        \"clusters\": [\"${cluster_id}\"]
    }" > /dev/null 2>&1
    ok "${ACS_SCAN_1}  (ocp4-stig, weekly Mon 04:00)"

    # Scan 2: CIS audit — same name as SSB_1 → deliberate conflict.
    acs_api POST "/v2/compliance/scan/configurations" -d "{
        \"scanName\": \"${ACS_SCAN_2}\",
        \"scanConfig\": {
            \"oneTimeScan\": false,
            \"profiles\": [\"ocp4-cis\"],
            \"scanSchedule\": {
                \"intervalType\": \"WEEKLY\",
                \"hour\": 6, \"minute\": 30,
                \"daysOfWeek\": { \"days\": [5] }
            },
            \"description\": \"Demo seed ${id}: CIS audit — pre-existing, will conflict with SSB\"
        },
        \"clusters\": [\"${cluster_id}\"]
    }" > /dev/null 2>&1
    ok "${ACS_SCAN_2}  (ocp4-cis, weekly Fri 06:30)  ← conflicts with SSB"

    # ── Save state ───────────────────────────────────────────────────────
    echo "$id" > "$STATE_FILE"

    hdr "Summary"
    echo -e "  ${BOLD}Seed ID:${RESET}    ${CYAN}${id}${RESET}"
    echo -e "  ${BOLD}K8s SSBs:${RESET}   ${SSB_1}, ${SSB_2}, ${SSB_3}"
    echo -e "  ${BOLD}ACS scans:${RESET}  ${ACS_SCAN_1}, ${ACS_SCAN_2}"
    echo -e "  ${BOLD}Conflict:${RESET}   ${RED}${SSB_1}${RESET} (SSB) vs ${RED}${ACS_SCAN_2}${RESET} (ACS)"
    echo ""
    echo -e "  ${DIM}Run the importer to see conflict handling:${RESET}"
    echo -e "  ${DIM}  ./compliance-operator-importer --endpoint \$ROX_ENDPOINT --insecure-skip-verify${RESET}"
    echo -e "  ${DIM}  ./compliance-operator-importer --endpoint \$ROX_ENDPOINT --insecure-skip-verify --overwrite-existing${RESET}"
    echo ""
    echo -e "  ${DIM}Tear down:  $0 down${RESET}"
    echo ""
}

# ── DOWN ─────────────────────────────────────────────────────────────────────

cmd_down() {
    local id
    id=$(load_id)
    names_for "$id"

    echo -e "${BOLD}Removing demo fixtures  [id: ${CYAN}${id}${RESET}${BOLD}]${RESET}"

    # ── K8s ──────────────────────────────────────────────────────────────
    hdr "Kubernetes"
    for name in "$SSB_1" "$SSB_2" "$SSB_3"; do
        if kubectl delete scansettingbinding "$name" -n "$CO_NS" --ignore-not-found 2> /dev/null; then
            ok "Deleted SSB ${name}"
        fi
    done
    if kubectl delete scansetting "$SCAN_SETTING" -n "$CO_NS" --ignore-not-found 2> /dev/null; then
        ok "Deleted ScanSetting ${SCAN_SETTING}"
    fi

    # ── ACS ──────────────────────────────────────────────────────────────
    hdr "ACS"
    local configs
    configs=$(acs_api GET "/v2/compliance/scan/configurations?pagination.limit=1000" 2> /dev/null)

    # Delete any scan config whose name starts with "demo-{id}-".
    echo "$configs" | python3 -c "
import sys, json
data = json.load(sys.stdin)
prefix = 'demo-${id}-'
for c in data.get('configurations', []):
    if c['scanName'].startswith(prefix):
        print(c['id'] + ' ' + c['scanName'])
" 2> /dev/null | while read -r cfg_id cfg_name; do
        acs_api DELETE "/v2/compliance/scan/configurations/${cfg_id}" > /dev/null 2>&1
        ok "Deleted ACS scan config ${cfg_name} (${cfg_id})"
    done

    rm -f "$STATE_FILE"
    echo ""
    ok "All demo-${id} fixtures removed."
    echo ""
}

# ── STATUS ───────────────────────────────────────────────────────────────────

cmd_status() {
    local id
    id=$(load_id)
    names_for "$id"

    echo -e "${BOLD}Demo fixtures status  [id: ${CYAN}${id}${RESET}${BOLD}]${RESET}"

    hdr "Kubernetes (namespace: ${CO_NS})"
    kubectl get scansettingbindings.compliance.openshift.io,scansettings.compliance.openshift.io \
        -n "$CO_NS" -l "demo-seed=${id}" \
        -o custom-columns='KIND:.kind,NAME:.metadata.name' --no-headers 2> /dev/null \
        | while read -r kind name; do
            info "${kind}: ${name}"
        done

    hdr "ACS"
    local configs
    configs=$(acs_api GET "/v2/compliance/scan/configurations?pagination.limit=1000" 2> /dev/null)
    echo "$configs" | python3 -c "
import sys, json
data = json.load(sys.stdin)
prefix = 'demo-${id}-'
for c in data.get('configurations', []):
    if c['scanName'].startswith(prefix):
        sched = c.get('scanConfig', {}).get('scanSchedule', {})
        profiles = c.get('scanConfig', {}).get('profiles', [])
        interval = sched.get('intervalType', '?')
        hour = sched.get('hour', '?')
        minute = sched.get('minute', 0)
        print(f\"  {c['scanName']}  ({', '.join(profiles)}, {interval} {hour}:{minute:02d})  id={c['id']}\")
" 2> /dev/null
    echo ""
}

# ── Main ─────────────────────────────────────────────────────────────────────
function help {
    echo "Usage: $0 {up|down|status}"
    echo ""
    echo "  up      Create 2 ACS scan configs + 3 K8s SSBs (1 conflicting)"
    echo "  down    Remove all fixtures created by 'up'"
    echo "  status  Show current fixture state"
}

case "${1:-}" in
    up) cmd_up "${2:-}" ;;
    down) cmd_down ;;
    status) cmd_status ;;
    help) help ;;
    -h) help ;;
    --help) help ;;
    *) cmd_up "${2:-}" ;;
esac
