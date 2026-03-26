#!/usr/bin/env bash
# demo.sh — Interactive demo of the CO → ACS scheduled scan importer.
#
# Prerequisites:
#   - kubectl configured with at least one context pointing to an OCP cluster
#     with the Compliance Operator installed
#   - ACS Central reachable from this machine
#   - ROX_ADMIN_PASSWORD or ROX_API_TOKEN set
#   - ROX_ENDPOINT set (or passed via --endpoint)
#   - The importer binary built:
#       cd scripts/compliance-operator-importer && go build -o compliance-operator-importer ./cmd/importer
#
# Usage:
#   ROX_ADMIN_PASSWORD=admin ROX_ENDPOINT=central.example.com ./demo.sh
#
# Non-interactive mode (for CI/testing):
#   DEMO_AUTO=1 ROX_ADMIN_PASSWORD=admin ROX_ENDPOINT=central.example.com ./demo.sh
#   DEMO_AUTO=1 DEMO_PAUSE=0 ...  # no pauses at all

set -euo pipefail

# ─────────────────────────────────────────────────────────────────────────────
# Configuration
# ─────────────────────────────────────────────────────────────────────────────

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMPORTER="${SCRIPT_DIR}/../compliance-operator-importer"
CO_NS="openshift-compliance"

# Resolve ACS endpoint — strip scheme, the importer adds it back.
ACS_ENDPOINT="${ROX_ENDPOINT:?ROX_ENDPOINT must be set}"
ACS_URL="https://${ACS_ENDPOINT#https://}"
ACS_URL="${ACS_URL#http://}"
ACS_URL="https://${ACS_URL#https://}"

# Auth for curl calls (basic auth only for this demo).
CURL_AUTH=(-u "admin:${ROX_ADMIN_PASSWORD:?ROX_ADMIN_PASSWORD must be set}")

# Importer flags.
IMPORTER_FLAGS=(--endpoint "$ACS_ENDPOINT" --insecure-skip-verify)

# Demo SSB names — prefixed to avoid collisions with real workloads.
DEMO_PREFIX="demo-import"
SSB_CIS="${DEMO_PREFIX}-cis-scan"
SSB_MODERATE="${DEMO_PREFIX}-moderate-scan"
SSB_PCI="${DEMO_PREFIX}-pci-dss-scan"

# ─────────────────────────────────────────────────────────────────────────────
# Helpers
# ─────────────────────────────────────────────────────────────────────────────

# Terminal colours.
BOLD='\033[1m'
DIM='\033[2m'
CYAN='\033[36m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
MAGENTA='\033[35m'
RESET='\033[0m'

banner() {
    local width=72
    echo ""
    echo -e "${CYAN}${BOLD}$(printf '═%.0s' $(seq 1 $width))${RESET}"
    echo "$1"
    echo -e "${CYAN}${BOLD}$(printf '═%.0s' $(seq 1 $width))${RESET}"
    echo ""
}

section() {
    echo ""
    echo -e "${MAGENTA}${BOLD}── $1 ──${RESET}"
    echo ""
}

info() {
    echo -e "${DIM}$1${RESET}"
}

narrate() {
    echo -e "${YELLOW}$1${RESET}"
}

success() {
    echo -e "${GREEN}  ✓ $1${RESET}"
}

fail_msg() {
    echo -e "${RED}  ✗ $1${RESET}"
}

pause() {
    echo ""
    if [[ "${DEMO_AUTO:-}" == "1" ]]; then
        sleep "${DEMO_PAUSE:-2}"
    else
        echo -ne "${DIM}Press ENTER to continue...${RESET}"
        read -r
    fi
    echo ""
}

run_cmd() {
    echo -e "${BOLD}\$ $*${RESET}"
    "$@" 2>&1 || true
    echo ""
}

acs_api() {
    local method="$1" path="$2"
    shift 2
    curl -sk "${CURL_AUTH[@]}" -X "$method" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        "${ACS_URL}${path}" "$@"
}

# ─────────────────────────────────────────────────────────────────────────────
# Cleanup helper — removes all demo resources
# ─────────────────────────────────────────────────────────────────────────────

cleanup_demo_resources() {
    local quiet="${1:-}"

    [[ -z "$quiet" ]] && info "Cleaning up demo resources..."

    # Delete demo SSBs from the cluster.
    for ssb in "$SSB_CIS" "$SSB_MODERATE" "$SSB_PCI"; do
        kubectl delete scansettingbinding "$ssb" -n "$CO_NS" --ignore-not-found 2> /dev/null || true
    done

    # Delete demo ScanSettings (original + ACS-created ones named after SSBs).
    kubectl delete scansetting "${DEMO_PREFIX}-setting" -n "$CO_NS" --ignore-not-found 2> /dev/null || true
    for ssb in "$SSB_CIS" "$SSB_MODERATE" "$SSB_PCI"; do
        kubectl delete scansetting "$ssb" -n "$CO_NS" --ignore-not-found 2> /dev/null || true
    done

    # Delete demo scan configs from ACS.
    local configs
    configs=$(acs_api GET "/v2/compliance/scan/configurations?pagination.limit=1000" 2> /dev/null)
    for ssb in "$SSB_CIS" "$SSB_MODERATE" "$SSB_PCI"; do
        local config_id
        config_id=$(echo "$configs" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for c in data.get('configurations', []):
    if c['scanName'] == '$ssb':
        print(c['id'])
        break
" 2> /dev/null || true)
        if [[ -n "$config_id" ]]; then
            acs_api DELETE "/v2/compliance/scan/configurations/$config_id" > /dev/null 2>&1 || true
        fi
    done

    [[ -z "$quiet" ]] && success "Done"
    return 0
}

# ─────────────────────────────────────────────────────────────────────────────
# Trap — clean up on exit or interrupt
# ─────────────────────────────────────────────────────────────────────────────

trap 'echo ""; cleanup_demo_resources' EXIT

# ═════════════════════════════════════════════════════════════════════════════
#  DEMO START
# ═════════════════════════════════════════════════════════════════════════════

clear
banner "CO → ACS Scheduled Scan Importer — Interactive Demo"

narrate "This demo walks through the importer tool that reads Compliance Operator"
narrate "ScanSettingBinding resources from Kubernetes and creates equivalent scan"
narrate "configurations in Red Hat Advanced Cluster Security (ACS)."
echo ""
narrate "We will:"
narrate "  1. Create demo ScanSettingBindings on the cluster"
narrate "  2. Run the importer in dry-run mode"
narrate "  3. Run the importer for real (happy path)"
narrate "  4. Run again to see skip behaviour (idempotency)"
narrate "  5. Simulate schedule drift on the Kubernetes side"
narrate "  6. Run without --overwrite-existing (drift preserved)"
narrate "  7. Run with --overwrite-existing (drift resolved)"
echo ""
info "Cluster:  $(kubectl config current-context)"
info "ACS:      $ACS_URL"
info "CO NS:    $CO_NS"

pause

# Pre-clean: silently remove leftovers from a previous run.
cleanup_demo_resources quiet

# ─────────────────────────────────────────────────────────────────────────────
#  STEP 1: Create demo ScanSetting and ScanSettingBindings
# ─────────────────────────────────────────────────────────────────────────────

banner "Step 1: Create Demo Resources"

narrate "First, we create a ScanSetting with a daily schedule (02:00 UTC),"
narrate "then three ScanSettingBindings that reference it — each binding"
narrate "targets a different compliance profile."

pause

section "Creating ScanSetting: ${DEMO_PREFIX}-setting"
info "Schedule: 0 2 * * * (daily at 02:00)"

run_cmd kubectl apply -f - << EOF
apiVersion: compliance.openshift.io/v1alpha1
kind: ScanSetting
metadata:
  name: ${DEMO_PREFIX}-setting
  namespace: ${CO_NS}
schedule: "0 2 * * *"
roles:
  - worker
  - master
rawResultStorage:
  rotation: 3
  size: 1Gi
EOF

section "Creating ScanSettingBinding: ${SSB_CIS}"
info "Profile: ocp4-cis"

run_cmd kubectl apply -f - << EOF
apiVersion: compliance.openshift.io/v1alpha1
kind: ScanSettingBinding
metadata:
  name: ${SSB_CIS}
  namespace: ${CO_NS}
profiles:
  - name: ocp4-cis
    kind: Profile
    apiGroup: compliance.openshift.io/v1alpha1
settingsRef:
  name: ${DEMO_PREFIX}-setting
  kind: ScanSetting
  apiGroup: compliance.openshift.io/v1alpha1
EOF

section "Creating ScanSettingBinding: ${SSB_MODERATE}"
info "Profile: ocp4-moderate"

run_cmd kubectl apply -f - << EOF
apiVersion: compliance.openshift.io/v1alpha1
kind: ScanSettingBinding
metadata:
  name: ${SSB_MODERATE}
  namespace: ${CO_NS}
profiles:
  - name: ocp4-moderate
    kind: Profile
    apiGroup: compliance.openshift.io/v1alpha1
settingsRef:
  name: ${DEMO_PREFIX}-setting
  kind: ScanSetting
  apiGroup: compliance.openshift.io/v1alpha1
EOF

section "Creating ScanSettingBinding: ${SSB_PCI}"
info "Profile: ocp4-pci-dss"

run_cmd kubectl apply -f - << EOF
apiVersion: compliance.openshift.io/v1alpha1
kind: ScanSettingBinding
metadata:
  name: ${SSB_PCI}
  namespace: ${CO_NS}
profiles:
  - name: ocp4-pci-dss
    kind: Profile
    apiGroup: compliance.openshift.io/v1alpha1
settingsRef:
  name: ${DEMO_PREFIX}-setting
  kind: ScanSetting
  apiGroup: compliance.openshift.io/v1alpha1
EOF

section "Verify: resources on the cluster"
run_cmd kubectl get scansettingbindings.compliance.openshift.io -n "$CO_NS" \
    -l '!app.kubernetes.io/managed-by' \
    -o custom-columns='NAME:.metadata.name,SETTING:.settingsRef.name,PROFILES:.profiles[*].name'

narrate "Three ScanSettingBindings created, each referencing the demo ScanSetting."
narrate "The importer will read these and create matching ACS scan configurations."

pause

# ─────────────────────────────────────────────────────────────────────────────
#  STEP 2: Dry run
# ─────────────────────────────────────────────────────────────────────────────

banner "Step 2: Dry Run"

narrate "Before making any changes, let's preview what the importer would do."
narrate "The --dry-run flag shows planned actions without touching ACS."

pause

run_cmd "$IMPORTER" "${IMPORTER_FLAGS[@]}" --dry-run

narrate "The importer discovered our 3 demo SSBs, mapped them to ACS scan"
narrate "configurations, and reported that it would create all three."
narrate "No changes were made to ACS."

pause

# ─────────────────────────────────────────────────────────────────────────────
#  STEP 3: Happy path — real import
# ─────────────────────────────────────────────────────────────────────────────

banner "Step 3: Import (Happy Path)"

narrate "Now let's run the importer for real. It will create three scan"
narrate "configurations in ACS, one for each ScanSettingBinding."

pause

run_cmd "$IMPORTER" "${IMPORTER_FLAGS[@]}"

section "Verify: scan configurations in ACS"
info "Querying ACS API for our demo scan configs..."
echo ""

for ssb in "$SSB_CIS" "$SSB_MODERATE" "$SSB_PCI"; do
    local_configs=$(acs_api GET "/v2/compliance/scan/configurations?pagination.limit=1000" 2> /dev/null)
    found=$(echo "$local_configs" | python3 -c "
import sys, json
data = json.load(sys.stdin)
for c in data.get('configurations', []):
    if c['scanName'] == '$ssb':
        sched = c.get('scanConfig', {}).get('scanSchedule', {})
        profiles = c.get('scanConfig', {}).get('profiles', [])
        print(f\"  Name:     {c['scanName']}\")
        print(f\"  ID:       {c['id']}\")
        print(f\"  Schedule: {sched.get('intervalType','?')} at {sched.get('hour','?')}:{sched.get('minute','?'):02d}\")
        print(f\"  Profiles: {', '.join(profiles)}\")
        break
" 2> /dev/null || true)
    if [[ -n "$found" ]]; then
        success "Found in ACS:"
        echo "$found"
        echo ""
    fi
done

narrate "All three scan configurations were created successfully in ACS."

pause

# ─────────────────────────────────────────────────────────────────────────────
#  STEP 4: Idempotency — run again, expect skips
# ─────────────────────────────────────────────────────────────────────────────

banner "Step 4: Idempotency"

narrate "What happens if we run the importer again? Since the scan configurations"
narrate "already exist in ACS, the importer should skip them gracefully."

pause

run_cmd "$IMPORTER" "${IMPORTER_FLAGS[@]}"

narrate "All three were skipped — the importer is idempotent by default."
narrate "It detects existing scan configs by name and does not create duplicates."

pause

# ─────────────────────────────────────────────────────────────────────────────
#  STEP 5: Simulate schedule drift on the Kubernetes side
# ─────────────────────────────────────────────────────────────────────────────

banner "Step 5: Simulate Schedule Drift"

narrate "After the initial import, each SSB was adopted — its settingsRef now"
narrate "points to an ACS-managed ScanSetting (same name as the scan config)."
narrate ""
narrate "Let's simulate a real-world scenario: someone edits the ACS-managed"
narrate "ScanSetting directly on the cluster (e.g. via kubectl).  ACS does NOT"
narrate "detect this change — the UI still shows the original schedule, but"
narrate "scans actually run on the new schedule.  A silent drift."

pause

section "Editing ACS-managed ScanSetting directly on the cluster"
info "ScanSetting '${SSB_CIS}' was created by ACS with schedule 0 2 * * *"
info "Patching it to 0 5 * * * (daily at 05:00)"

run_cmd kubectl patch scansetting "${SSB_CIS}" -n "$CO_NS" \
    --type merge -p '{"schedule": "0 5 * * *"}'

section "Verify: cluster vs ACS"
echo ""
echo -e "${BOLD}On the cluster (actual behaviour):${RESET}"
kubectl get scansetting "${SSB_CIS}" -n "$CO_NS" \
    -o custom-columns='SCANSETTING:.metadata.name,SCHEDULE:.schedule' --no-headers
echo ""
echo -e "${BOLD}In ACS (what the UI shows):${RESET}"
acs_api GET "/v2/compliance/scan/configurations?pagination.limit=1000" 2> /dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
for c in data.get('configurations', []):
    if c['scanName'] == '${SSB_CIS}':
        sched = c.get('scanConfig', {}).get('scanSchedule', {})
        print(f\"  {c['scanName']}: {sched.get('intervalType','?')} at {sched.get('hour','?')}:{sched.get('minute','?'):02d}\")
        break
" 2> /dev/null
echo ""

narrate "The cluster now scans at 05:00, but ACS still thinks it's 02:00."
narrate "This silent drift is exactly what the importer can detect and fix."

pause

# ─────────────────────────────────────────────────────────────────────────────
#  STEP 6: Run without --overwrite-existing (skip conflict)
# ─────────────────────────────────────────────────────────────────────────────

banner "Step 6: Default Behaviour (Skip Conflicts)"

narrate "Running the importer without --overwrite-existing. The scan config"
narrate "already exists in ACS, so the importer will skip it — even though"
narrate "the schedule has drifted on the cluster."

pause

run_cmd "$IMPORTER" "${IMPORTER_FLAGS[@]}"

narrate "All three were skipped — the importer found existing configs by name"
narrate "and left them untouched. The drifted CIS config was NOT updated."
narrate "This is the safe default: no surprises, no overwrites."

pause

# ─────────────────────────────────────────────────────────────────────────────
#  STEP 7: Run with --overwrite-existing (resolve drift)
# ─────────────────────────────────────────────────────────────────────────────

banner "Step 7: Overwrite Mode (Resolve Drift)"

narrate "Now let's run with --overwrite-existing. This tells the importer to"
narrate "update existing ACS scan configs to match what's on the cluster."
narrate "The CIS config in ACS will be updated from 02:00 → 05:00."

pause

run_cmd "$IMPORTER" "${IMPORTER_FLAGS[@]}" --overwrite-existing

section "Verify: ACS now matches the cluster"
acs_api GET "/v2/compliance/scan/configurations?pagination.limit=1000" 2> /dev/null | python3 -c "
import sys, json
data = json.load(sys.stdin)
for c in data.get('configurations', []):
    if c['scanName'] == '${SSB_CIS}':
        sched = c.get('scanConfig', {}).get('scanSchedule', {})
        print(f\"  Name:        {c['scanName']}\")
        print(f\"  Schedule:    {sched.get('intervalType','?')} at {sched.get('hour','?')}:{sched.get('minute','?'):02d}\")
        break
" 2> /dev/null
echo ""

narrate "The CIS scan config has been updated to DAILY 05:00 — matching the"
narrate "cluster's ScanSetting. The --overwrite-existing flag ensures ACS"
narrate "stays in sync with the Compliance Operator source of truth."

pause

# ─────────────────────────────────────────────────────────────────────────────
#  Done — EXIT trap handles cleanup automatically
# ─────────────────────────────────────────────────────────────────────────────

banner "Demo Complete"

narrate "Summary of what we demonstrated:"
echo ""
echo -e "  ${GREEN}1.${RESET} Created CO resources (ScanSetting + 3 ScanSettingBindings)"
echo -e "  ${GREEN}2.${RESET} Dry-run mode: preview without side effects"
echo -e "  ${GREEN}3.${RESET} Happy path: imported all SSBs into ACS scan configs + adoption"
echo -e "  ${GREEN}4.${RESET} Idempotency: re-run skips existing configs safely"
echo -e "  ${GREEN}5.${RESET} Schedule drift: changed ScanSetting schedule on the cluster"
echo -e "  ${GREEN}6.${RESET} Default skip: drift preserved without --overwrite-existing"
echo -e "  ${GREEN}7.${RESET} Overwrite mode: drift resolved, ACS re-synced to cluster"
echo ""

# The EXIT trap handles cleanup automatically.
