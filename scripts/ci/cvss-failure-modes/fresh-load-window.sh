#!/usr/bin/env bash
# Scenario: fresh-load-window (mechanism C)
# Measures the window between the matcher reporting Ready (Initialized) and
# scans actually returning vulnerability data, on a CLEAN initial load with no
# interruption. Every probe sample is correlated with Central's log so that
# "empty" scans are attributed to enrichment errors (registry 401, indexer,
# matcher errors) vs genuinely empty data - the confound that hid this window
# in earlier measurements.
#
# Verdict: FAIL if the Ready -> usable window exceeds WINDOW_THRESHOLD_SECONDS
# (default 120). The full timeline is printed either way.

set -euo pipefail
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SOURCE_DIR}/lib.sh"

WINDOW_THRESHOLD_SECONDS="${WINDOW_THRESHOLD_SECONDS:-120}"

cvss_log "=== fresh-load-window: waiting for the matcher pod ==="
for i in $(seq 1 120); do
  kubectl -n "$NAMESPACE" get pods -l app=scanner-v4-matcher \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null | grep -q . && break
  sleep 5
done

cvss_log "=== probing until scans recover (this covers the whole initial load) ==="
ready_at="" first_good_at="" verdict="INCONCLUSIVE"
deadline=$((SECONDS + 2700))
while (( SECONDS < deadline )); do
  # Matcher Ready timestamp (once).
  if [[ -z "$ready_at" ]] && kubectl -n "$NAMESPACE" get pods -l app=scanner-v4-matcher \
      -o jsonpath='{.items[0].status.containerStatuses[0].ready}' 2>/dev/null | grep -q true; then
    ready_at="$(date -u +%H:%M:%S)"
    ready_secs=$SECONDS
    cvss_log "MATCHER READY at ${ready_at}"
  fi
  cvss_probe_once
  cvss_matcher_milestones 15s || true
  last="$(cvss_scan_summary true)"
  if [[ -n "$ready_at" && "$last" =~ maxCvss=([7-9]|[1-9][0-9]) ]]; then
    first_good_at="$(date -u +%H:%M:%S)"
    break
  fi
  sleep 10
done

if [[ -z "$first_good_at" ]]; then
  cvss_log "VERDICT: INCONCLUSIVE - scans never recovered within budget"
  exit 1
fi
window=$((SECONDS - ready_secs))
cvss_log "=== TIMELINE: Ready=${ready_at} first-good=${first_good_at} window=${window}s ==="
cvss_log "NOTE: see PROBE lines above - samples with enrich_errors>0 were scan-pipeline"
cvss_log "failures (Central served the stale stored scan); only enrich_errors=0 samples"
cvss_log "reflect actual matcher data state."

if (( window > WINDOW_THRESHOLD_SECONDS )); then
  cvss_log "VERDICT: FAIL - Ready-but-unusable window (${window}s) exceeds ${WINDOW_THRESHOLD_SECONDS}s."
  cvss_dump_forensics
  exit 1
fi
cvss_log "VERDICT: PASS - window ${window}s within threshold."
