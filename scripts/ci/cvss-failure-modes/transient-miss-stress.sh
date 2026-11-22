#!/usr/bin/env bash
# Scenario: transient-miss-stress (mechanism D)
# Detects the transient "0 alerts with full data" miss observed in CI (~1-2%
# of detectBuildTime calls): a build-time detection request returns no
# vulnerability-policy alerts while the stored image data is complete and a
# control (non-vuln) policy does fire, and an immediate retry succeeds.
#
# Runs detectBuildTime (POST /v1/detect/build) in a loop against a healthy
# deployment, with a background probe hammering scans every 10s to provide
# the concurrency under which the miss was observed.
#
# Verdict: FAIL on any miss (with forensics: other policies that fired, retry
# result, call duration). PASS if all iterations are clean.

set -euo pipefail
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${SOURCE_DIR}/lib.sh"

ITERATIONS="${ITERATIONS:-60}"
DELAY_SECONDS="${DELAY_SECONDS:-10}"
CVSS_POLICY="Fixable CVSS >= 7"

cvss_log "=== transient-miss-stress: waiting for a healthy scan ==="
deadline=$((SECONDS + 2700))
while (( SECONDS < deadline )); do
  last="$(cvss_scan_summary true)"
  [[ "$last" =~ maxCvss=([7-9]|[1-9][0-9]) ]] && break
  sleep 10
done
[[ "$last" =~ maxCvss=([7-9]|[1-9][0-9]) ]] || { cvss_log "ERROR: never healthy"; exit 1; }
cvss_log "healthy baseline: ${last}"

cvss_log "=== starting background probe (concurrency) ==="
( while true; do cvss_scan_summary true >/dev/null 2>&1 || true; sleep 10; done ) &
PROBE_PID=$!
trap 'kill $PROBE_PID 2>/dev/null || true' EXIT

detect_body='{"image":{"name":{"registry":"quay.io","remote":"rhacs-eng/qa-multi-arch","tag":"nginx-2.0.3"}}}'
misses=0
for ((i = 1; i <= ITERATIONS; i++)); do
  t0=$(date +%s%3N)
  resp="$(cvss_api -H 'Content-Type: application/json' -d "$detect_body" \
    "https://${API_ENDPOINT}/v1/detect/build" 2>/dev/null || echo '{}')"
  t1=$(date +%s%3N)
  call_ms=$((t1 - t0))
  cvss_alerts="$(jq -r --arg p "$CVSS_POLICY" '[.alerts[]? | select(.policy.name == $p)] | length' <<<"$resp" 2>/dev/null || echo -1)"
  other="$(jq -r '[.alerts[]?.policy.name] | unique | join(",")' <<<"$resp" 2>/dev/null || echo "?")"
  if [[ "$cvss_alerts" == "0" && "$other" != "" ]]; then
    # Miss: CVSS policy absent while other policies fired. Immediate retry:
    r0=$(date +%s%3N)
    retry="$(cvss_api -H 'Content-Type: application/json' -d "$detect_body" \
      "https://${API_ENDPOINT}/v1/detect/build" 2>/dev/null || echo '{}')"
    r1=$(date +%s%3N)
    retry_alerts="$(jq -r --arg p "$CVSS_POLICY" '[.alerts[]? | select(.policy.name == $p)] | length' <<<"$retry" 2>/dev/null || echo -1)"
    stored="$(cvss_scan_summary false)"
    cvss_log "MISS iter=${i} callMs=${call_ms} otherPolicies=[${other}] retryAlerts=${retry_alerts} retryMs=$((r1 - r0)) stored=(${stored})"
    misses=$((misses + 1))
  else
    cvss_log "iter=${i}/${ITERATIONS} cvssAlerts=${cvss_alerts} callMs=${call_ms} totalAlerts=$(jq -r '.alerts | length' <<<"$resp" 2>/dev/null || echo '?')"
  fi
  sleep "$DELAY_SECONDS"
done

if (( misses > 0 )); then
  cvss_log "VERDICT: FAIL - ${misses}/${ITERATIONS} detectBuildTime calls missed the"
  cvss_log "vulnerability policy with full stored data. See MISS lines for forensics."
  cvss_dump_forensics
  exit 1
fi
cvss_log "VERDICT: PASS - ${ITERATIONS} clean iterations."
