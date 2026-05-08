#!/usr/bin/env bash
# bench-reassess.sh -- E2E benchmark: burst → reassess → burst cycle.
# Collects AC, Sensor, and Central Prometheus metrics to quantify
# reprocessor cache optimization impact. See README.md for details.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------

: "${BURST_SIZE:=100}"
: "${UNIQUE_PCT:=25}"
: "${PARALLEL:=50}"
: "${NAMESPACE:=burst-test}"
: "${ROX_NAMESPACE:=stackrox}"
: "${ROX_PASSWORD:?ROX_PASSWORD must be set}"
: "${ROX_CENTRAL_ADDRESS:?ROX_CENTRAL_ADDRESS must be set}"
: "${ROX_ADMIN_USER:=admin}"
: "${REASSESS_WAIT_TIMEOUT:=300}"

: "${METRICS_PORT:=9090}"
: "${LOCAL_PORT:=9090}"

DRY_RUN="true"
export DRY_RUN

WORK_DIR=$(mktemp -d)

# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"

# component-label → local-port mapping (all use the same local port, scraped sequentially)
declare -A COMPONENTS=(
    [admission-control]="$LOCAL_PORT"
    [sensor]="$LOCAL_PORT"
    [central]="$LOCAL_PORT"
)
declare -A COMP_SHORT=( [admission-control]=ac [sensor]=sensor [central]=central )

readonly NETPOL_PREFIX="bench-reassess-metrics"
trap cleanup EXIT

# ---------------------------------------------------------------------------
# bench-reassess-specific helpers
# ---------------------------------------------------------------------------

cleanup() {
    kill_port_forwards
    for app_label in admission-control sensor central; do
        kubectl -n "$ROX_NAMESPACE" delete networkpolicy "${NETPOL_PREFIX}-${app_label}" \
            --ignore-not-found &>/dev/null || true
    done
    kubectl delete namespace "$NAMESPACE" --ignore-not-found &>/dev/null || true
    [[ -n "${WORK_DIR:-}" ]] && rm -rf "$WORK_DIR"
}

allow_metrics_ingress() {
    for app_label in admission-control sensor central; do
        allow_metrics_netpol "${NETPOL_PREFIX}-${app_label}" "$app_label"
    done
}

scrape_all() {
    local prefix="$1"
    for label in "${!COMPONENTS[@]}"; do
        local short="${COMP_SHORT[$label]}"
        echo "  Scraping ${short} metrics..."
        scrape_component "$label" "${COMPONENTS[$label]}" \
            "${WORK_DIR}/${prefix}_${short}.prom"
    done
}

# read_metric <phase> <component-short> <metric> [labels]
read_metric() {
    extract "${WORK_DIR}/${1}_${2}.prom" "$3" "${4:-}"
}

# metric_delta <phase_a> <phase_b> <component-short> <metric> [labels]
metric_delta() {
    echo "$(read_metric "$2" "$3" "$4" "${5:-}") - $(read_metric "$1" "$3" "$4" "${5:-}")" | bc
}

# trigger_and_wait_for_reprocessor is now in lib.sh

scrape_reprocessor_duration() {
    echo "  Scraping reprocessor duration gauge..."
    sleep 3
    scrape_component "central" "${COMPONENTS[central]}" "${WORK_DIR}/poll_central.prom"
    local duration
    duration=$(extract "${WORK_DIR}/poll_central.prom" "rox_central_reprocessor_duration_seconds")
    echo "  reprocessor_duration_seconds = ${duration:-N/A}"
}

# ---------------------------------------------------------------------------
# Report
# ---------------------------------------------------------------------------

print_report() {
    local pre="$1" post="$2" post2="$3"
    local ac="rox_admission_control" s="rox_sensor" c="rox_central"

    # AC burst 2 (post → post2)
    local b2_hit b2_miss b2_skip b2_fetch b2_dur_avg
    b2_hit=$(metric_delta "$post" "$post2" ac "${ac}_image_cache_operations_total" 'result="hit"')
    b2_miss=$(metric_delta "$post" "$post2" ac "${ac}_image_cache_operations_total" 'result="miss"')
    b2_skip=$(metric_delta "$post" "$post2" ac "${ac}_image_cache_operations_total" 'result="skip"')
    b2_fetch=$(metric_delta "$post" "$post2" ac "${ac}_image_fetch_total")
    b2_dur_avg=$(avg_or_na \
        "$(metric_delta "$post" "$post2" ac "${ac}_policyeval_review_duration_seconds_sum")" \
        "$(metric_delta "$post" "$post2" ac "${ac}_policyeval_review_duration_seconds_count")")

    local b2_total b2_hit_pct
    b2_total=$(echo "${b2_hit} + ${b2_miss} + ${b2_skip}" | bc)
    b2_hit_pct=$(pct_or_na "$b2_hit" "$b2_total")

    # AC reassess (pre → post)
    local r_ac_hit r_ac_miss r_ac_skip r_ac_fetch
    r_ac_hit=$(metric_delta "$pre" "$post" ac "${ac}_image_cache_operations_total" 'result="hit"')
    r_ac_miss=$(metric_delta "$pre" "$post" ac "${ac}_image_cache_operations_total" 'result="miss"')
    r_ac_skip=$(metric_delta "$pre" "$post" ac "${ac}_image_cache_operations_total" 'result="skip"')
    r_ac_fetch=$(metric_delta "$pre" "$post" ac "${ac}_image_fetch_total")

    # Sensor reassess (pre → post)
    local r_s_depl r_s_dedupe r_s_events
    r_s_depl=$(metric_delta "$pre" "$post" sensor "${s}_detector_deployment_processed")
    r_s_dedupe=$(metric_delta "$pre" "$post" sensor "${s}_detector_dedupe_cache_hits")
    r_s_events=$(metric_delta "$pre" "$post" sensor "${s}_sensor_events")

    local r_s_cpm_count r_s_cpm_avg
    r_s_cpm_count=$(metric_delta "$pre" "$post" sensor "${s}_component_process_message_duration_seconds_count")
    r_s_cpm_avg=$(avg_or_na \
        "$(metric_delta "$pre" "$post" sensor "${s}_component_process_message_duration_seconds_sum")" \
        "$r_s_cpm_count")


    # Sensor burst 2 (post → post2)
    local b2_s_depl b2_s_dedupe b2_s_events
    b2_s_depl=$(metric_delta "$post" "$post2" sensor "${s}_detector_deployment_processed")
    b2_s_dedupe=$(metric_delta "$post" "$post2" sensor "${s}_detector_dedupe_cache_hits")
    b2_s_events=$(metric_delta "$post" "$post2" sensor "${s}_sensor_events")

    # Central
    local c_dur c_notsent
    c_dur=$(read_metric "$post" central "${c}_reprocessor_duration_seconds")
    c_notsent=$(metric_delta "$pre" "$post" central "${c}_msg_to_sensor_not_sent_count")

    # --- Output ---
    cat <<EOF

================================================================================
  BENCH-REASSESS REPORT
================================================================================
  Burst size:      ${BURST_SIZE}
  Unique images:   ${UNIQUE_COUNT} / ${POOL_SIZE}
  Parallelism:     ${PARALLEL}
  Central:         ${ROX_CENTRAL_ADDRESS}
================================================================================

  CENTRAL  (reassess cycle)
  --------------------------------
EOF
    row "reprocessor_duration_seconds (gauge)"  "$c_dur"
    row "msg_to_sensor_not_sent (delta)"        "$c_notsent"

    cat <<EOF

  SENSOR  (during reassess)
  --------------------------------
EOF
    row "detector_deployment_processed (delta)"   "$r_s_depl"
    row "detector_dedupe_cache_hits (delta)"       "$r_s_dedupe"
    row "sensor_events (delta)"                    "$r_s_events"
    row "component_process_message count (delta)"  "$r_s_cpm_count"
    row "component_process_message avg (s)"        "$r_s_cpm_avg"

    cat <<EOF

  ADMISSION CONTROLLER  (during reassess)
  --------------------------------
EOF
    row "image_cache_operations{hit} (delta)"   "$r_ac_hit"
    row "image_cache_operations{miss} (delta)"  "$r_ac_miss"
    row "image_cache_operations{skip} (delta)"  "$r_ac_skip"
    row "image_fetch_total (delta)"             "$r_ac_fetch"

    cat <<EOF

  ADMISSION CONTROLLER  (burst 2)
  --------------------------------
EOF
    row "image_cache_operations{hit} (delta)"   "$b2_hit"
    row "image_cache_operations{miss} (delta)"  "$b2_miss"
    row "image_cache_operations{skip} (delta)"  "$b2_skip"
    row "image_fetch_total (delta)"             "$b2_fetch"
    row "cache_hit_rate"                        "${b2_hit_pct}%"
    row "review_duration avg (s)"              "$b2_dur_avg"

    cat <<EOF

  SENSOR  (burst 2)
  --------------------------------
EOF
    row "detector_deployment_processed (delta)"  "$b2_s_depl"
    row "detector_dedupe_cache_hits (delta)"      "$b2_s_dedupe"
    row "sensor_events (delta)"                   "$b2_s_events"

    cat <<EOF

================================================================================
  KEY COMPARISONS  (run once on master, once on PR branch)
================================================================================
  1. Central reprocessor_duration_seconds:   ${c_dur}
     Lower = less time serializing/sending UpdatedImage messages.

  2. Sensor deployment_processed (reassess): ${r_s_depl}
     Lower = fewer ResolveDeploymentsByImages re-detections.

  3. AC cache hit rate on burst 2:           ${b2_hit_pct}%
     Higher = cache survived reassess (targeted invalidation).

  4. AC image_fetch_total (burst 2):         ${b2_fetch}
     Lower = fewer cold fetches (cache was warm).

  5. AC image_fetch_total (reassess):        ${r_ac_fetch}
     Lower = no unnecessary invalidation-triggered re-fetches.
================================================================================

EOF
}

# ---------------------------------------------------------------------------
# Main flow
# ---------------------------------------------------------------------------

cat <<EOF
================================================================================
  bench-reassess.sh
================================================================================
  BURST_SIZE:            ${BURST_SIZE}
  UNIQUE_PCT:            ${UNIQUE_PCT}%
  PARALLEL:              ${PARALLEL}
  ROX_CENTRAL_ADDRESS:   ${ROX_CENTRAL_ADDRESS}
  REASSESS_WAIT_TIMEOUT: ${REASSESS_WAIT_TIMEOUT}s
================================================================================

EOF

setup_namespace
allow_metrics_ingress

echo "--- Generating manifests ---"
generate_slow_path_manifests

echo ""
echo "=== PHASE 1: Burst 1 (warm caches) ==="
run_burst "Burst 1"
echo "  Waiting 10s for metrics to settle..."
sleep 10

echo ""
echo "=== PHASE 2: Snapshot pre-reassess metrics ==="
scrape_all "pre"

echo ""
echo "=== PHASE 3: Trigger reassess ==="
trigger_and_wait_for_reprocessor
scrape_reprocessor_duration

echo ""
echo "=== PHASE 4: Snapshot post-reassess metrics ==="
scrape_all "post"

echo ""
echo "=== PHASE 5: Burst 2 (measure cache survival) ==="
run_burst "Burst 2"
echo "  Waiting 10s for metrics to settle..."
sleep 10

echo ""
echo "=== PHASE 6: Snapshot post-burst2 metrics ==="
scrape_all "post2"

echo ""
echo "=== PHASE 7: Report ==="
print_report "pre" "post" "post2"
