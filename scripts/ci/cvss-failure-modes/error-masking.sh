#!/usr/bin/env bash
# Scenario: error-masking (mechanism E)
# Verifies how Central's scan API behaves when the scan pipeline is broken:
# with a good scan stored, the matcher is scaled to zero and a forced scan is
# requested. If the API returns the stale stored scan as if it were current,
# pipeline failures are silently masked as (stale) data - the behavior that
# made every failure mode in this investigation look like "empty data".
#
# Verdict: FAIL if the forced scan returns stale data without any error
# indication (masking). PASS if the failure is surfaced.

set -euo pipefail
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SOURCE_DIR}/lib.sh"

cvss_log "=== error-masking: waiting for a healthy scan ==="
deadline=$((SECONDS + 2700))
while (( SECONDS < deadline )); do
  last="$(cvss_scan_summary true)"
  [[ "$last" =~ maxCvss=([7-9]|[1-9][0-9]) ]] && break
  sleep 10
done
[[ "$last" =~ maxCvss=([7-9]|[1-9][0-9]) ]] || { cvss_log "ERROR: never healthy"; exit 1; }
cvss_log "healthy baseline: ${last}"

cvss_log "=== breaking the scan pipeline: scaling matcher to 0 ==="
kubectl -n "$NAMESPACE" scale deploy/scanner-v4-matcher --replicas=0
kubectl -n "$NAMESPACE" wait --for=delete pod -l app=scanner-v4-matcher --timeout=120s || true
sleep 5

cvss_log "=== forced scan with the pipeline broken ==="
masked_result="$(cvss_scan_summary true)"
errs="$(cvss_central_scan_log 60s | grep -c "Error: Enriching" || true)"
cvss_log "forced scan returned: ${masked_result}; enrichment errors logged: ${errs}"

cvss_log "=== restoring matcher ==="
kubectl -n "$NAMESPACE" scale deploy/scanner-v4-matcher --replicas=1 || true

if [[ "$masked_result" =~ maxCvss=([7-9]|[1-9][0-9]) ]]; then
  cvss_log "VERDICT: FAIL - MASKING CONFIRMED: the scan API returned the stale stored"
  cvss_log "scan (full data) while the matcher was down and enrichment was erroring"
  cvss_log "(errors only visible in Central's log, not in the API response)."
  cvss_log "Clients cannot distinguish stale data from current data."
  exit 1
elif [[ "$masked_result" == "scan_failed" ]]; then
  cvss_log "VERDICT: PASS - the scan failure was surfaced."
  exit 0
else
  cvss_log "VERDICT: PARTIAL - the API returned data (${masked_result}) with the pipeline"
  cvss_log "broken; check whether it is stale (masking) or empty (visible symptom)."
  exit 1
fi
