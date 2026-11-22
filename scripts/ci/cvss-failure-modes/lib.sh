#!/usr/bin/env bash
# Shared library for the ROX-36267 failure-mode test scenarios.
# Usage: source this file, then call its functions. Requires: kubectl, jq, curl,
# a deployed StackRox (see deploy_stackrox), and /tmp/repro-envrc from roxie.

FIXTURE_IMAGE="${FIXTURE_IMAGE:-quay.io/rhacs-eng/qa-multi-arch:nginx-2.0.3}"
NAMESPACE="${NAMESPACE:-stackrox}"

cvss_log() { echo "[$(date -u '+%H:%M:%S')] $*"; }

cvss_admin_password() {
  grep ROX_ADMIN_PASSWORD /tmp/repro-envrc | cut -d= -f2 | tr -d "'\""
}

cvss_api() {
  curl -sk --max-time 30 -u "admin:$(cvss_admin_password)" "$@"
}

# scan_summary <force> - one scan; prints "fixable=N maxCvss=M" or "scan_failed"
cvss_scan_summary() {
  local force="$1"
  cvss_api -H 'Content-Type: application/json' \
    -d "{\"imageName\": \"${FIXTURE_IMAGE}\", \"force\": ${force}}" \
    "https://${API_ENDPOINT}/v1/images/scan" 2>/dev/null | jq -r \
    '[.scan.components[]?.vulns[]? | select(.fixedBy != "")] as $f |
     "fixable=\($f|length) maxCvss=\([$f[].cvss] | max // 0)"' 2>/dev/null || echo "scan_failed"
}

# central_scan_log <since> - Central's scan-path log lines for correlation.
# Prints enrichment errors and matcher vuln-report lines from the last <since>.
cvss_central_scan_log() {
  local since="${1:-12s}"
  { kubectl -n "$NAMESPACE" logs deploy/central --since="$since" 2>/dev/null \
    | grep -E "Error: Enriching image|Vuln report" || true; } | tail -5
}

# probe_once - the instrumented sample: forced + unforced scan, correlated with
# Central's log so "empty" is attributed to an enrichment error vs empty data,
# and with the matcher's own report when debug logging is on.
cvss_probe_once() {
  local forced unforced logline errs report
  forced="$(cvss_scan_summary true)"
  unforced="$(cvss_scan_summary false)"
  logline="$(cvss_central_scan_log 12s || true)"
  errs="$(grep -c "Error: Enriching image" <<<"$logline" 2>/dev/null || true)"
  report="$(grep "Vuln report" <<<"$logline" 2>/dev/null | tail -1 \
    | grep -oE '[0-9]+ pkg vulns, [0-9]+ vulns' || true)"
  cvss_log "PROBE forced=(${forced}) unforced=(${unforced}) enrich_errors=${errs:-0} matcher_report=${report:-n/a}"
}

# matcher_msg <pod> <since> - recent matcher milestone log lines.
cvss_matcher_milestones() {
  local pod since="${1:-15s}"
  pod="$(kubectl -n "$NAMESPACE" get pods -l app=scanner-v4-matcher -o name | head -1)"
  [ -z "$pod" ] && return 0
  { kubectl -n "$NAMESPACE" logs "$pod" --since="$since" 2>/dev/null \
    | grep -E '"msg":"(starting update|completed bundle update|completed update|GC started|GC completed|setting to initialized|fingerprint match, skipping|importing update)"' || true; } \
    | python3 -c "
import json,sys
for line in sys.stdin:
    try: d=json.loads(line)
    except: continue
    print('MATCHER', d.get('time','')[11:19], d.get('msg','')[:60], d.get('bundle',''), d.get('updater',''))
"
}

# wait_matcher_ready - waits for the matcher pod Ready; prints seconds elapsed.
cvss_wait_matcher_ready() {
  local timeout="${1:-2400}" start=$SECONDS
  kubectl -n "$NAMESPACE" wait --for=jsonpath='{.status.containerStatuses[0].ready}'=true \
    pod -l app=scanner-v4-matcher --timeout="${timeout}s" >/dev/null 2>&1 \
    || { cvss_log "ERROR: matcher never became ready"; return 1; }
  cvss_log "matcher Ready after $((SECONDS - start))s"
}

cvss_dump_forensics() {
  cvss_log "############ MATCHER LOGS ############"
  for pod in $(kubectl -n "$NAMESPACE" get pods -l app=scanner-v4-matcher -o name); do
    echo "--- $pod:"
    kubectl -n "$NAMESPACE" logs "$pod" 2>/dev/null | tail -200 || true
  done
  cvss_log "############ CENTRAL SCAN LINES ############"
  kubectl -n "$NAMESPACE" logs deploy/central 2>/dev/null \
    | grep -E "Vuln report|Error: Enriching" | tail -40 || true
  cvss_log "############ SCANNER DB STATE ############"
  local dbpass
  dbpass="$(kubectl -n "$NAMESPACE" get secret scanner-v4-db-password \
    -o jsonpath='{.data.password}' 2>/dev/null | base64 -d || true)"
  if [[ -n "$dbpass" ]]; then
    kubectl -n "$NAMESPACE" exec deploy/scanner-v4-db -c db -- \
      env PGPASSWORD="$dbpass" psql -U postgres -h localhost -d postgres -c \
      "SELECT key, update_timestamp FROM last_vuln_update ORDER BY update_timestamp;
       SELECT u.id, u.updater, u.date,
              (SELECT count(*) FROM uo_enrich ue WHERE ue.uo = u.id) AS enrich_records,
              (SELECT count(*) FROM uo_vuln uv WHERE uv.uo = u.id) AS vuln_records
        FROM update_operation u ORDER BY u.date DESC LIMIT 30;" 2>&1 || true
  fi
  cvss_log "############ END FORENSICS ############"
}
