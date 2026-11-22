#!/usr/bin/env bash
# Scenario: interrupted-import (mechanism A)
# Kills the matcher pod mid-NVD-import (simulating OOM/rollout/node drain).
# The interrupted import leaves an update_operation with a fingerprint but no
# records; the replacement matcher skips re-import (fingerprint match on the
# empty UO) and reports Ready while scans lack vulnerability data.
#
# Verdict: PASS if scans recover after the interruption (a fix is present),
# FAIL if the matcher is Ready but scans stay empty (bug reproduced).

set -euo pipefail
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SOURCE_DIR}/lib.sh"

INTERRUPT_DELAY_SECONDS="${INTERRUPT_DELAY_SECONDS:-10}"

cvss_log "=== interrupted-import: waiting for NVD import to start ==="
matcher_pod=""
deadline=$((SECONDS + 2400))
while (( SECONDS < deadline )); do
  matcher_pod="$(kubectl -n "$NAMESPACE" get pods -l app=scanner-v4-matcher -o name | head -1 | cut -d/ -f2)"
  if [[ -n "$matcher_pod" ]] && kubectl -n "$NAMESPACE" logs "$matcher_pod" 2>/dev/null \
      | grep -q '"msg":"importing update".*"updater":"nvd"'; then
    break
  fi
  sleep 5
done
if ! kubectl -n "$NAMESPACE" logs "$matcher_pod" 2>/dev/null \
      | grep -q '"msg":"importing update".*"updater":"nvd"'; then
  cvss_log "ERROR: NVD import never started"; exit 1
fi
cvss_log "NVD importing on ${matcher_pod}; killing in ${INTERRUPT_DELAY_SECONDS}s"
sleep "$INTERRUPT_DELAY_SECONDS"
kubectl -n "$NAMESPACE" delete pod "$matcher_pod" --grace-period=0

cvss_wait_matcher_ready 2400

cvss_log "=== probing for recovery (up to 15 min) ==="
last_clean_zero=$SECONDS
verdict="INCONCLUSIVE"
deadline=$((SECONDS + 900))
while (( SECONDS < deadline )); do
  cvss_probe_once
  cvss_matcher_milestones 15s || true
  last="$(cvss_scan_summary true)"
  errs="$(cvss_central_scan_log 12s | grep -c "Error: Enriching" || true)"
  if [[ "$last" =~ maxCvss=([7-9]|[1-9][0-9]) ]]; then
    verdict="RECOVERED"; break
  fi
  # Count only clean (non-erroring) empty scans toward the bug threshold:
  # errored scans are a different failure mode (see error-masking scenario).
  if [[ "${errs:-0}" -eq 0 ]]; then
    last_clean_zero=$SECONDS
  fi
  if (( SECONDS - last_clean_zero > 720 )); then
    verdict="BUG_REPRODUCED"; break
  fi
  sleep 10
done

case "$verdict" in
  RECOVERED)
    cvss_log "VERDICT: PASS - scans recovered after interrupted import."; exit 0 ;;
  BUG_REPRODUCED)
    cvss_log "VERDICT: FAIL - BUG REPRODUCED: matcher Ready, scans clean-but-empty"
    cvss_log "for >12 min. Empty UO + fingerprint skip visible in forensics."
    cvss_dump_forensics; exit 1 ;;
  *)
    cvss_log "VERDICT: INCONCLUSIVE"; cvss_dump_forensics; exit 1 ;;
esac
