#!/usr/bin/env bash
# burst-test.sh -- Admission controller burst test rig.
#
# Simulates a burst of deployment creation requests (--dry-run=server) against
# a live cluster, scrapes admission controller Prometheus metrics before and
# after, and reports deltas.
#
# Two modes:
#   fast-path  - Tests spec-only policy evaluation (no image fetching).
#                Requires: Privileged Container + Latest tag policies enforced.
#   slow-path  - Tests image coalescing and caching in the enrichment path.
#                Runs two phases (cold cache, then warm cache).
#                Requires: at least one enrichment-required policy (e.g. Image Age).
#
# See README.md for full prerequisites and cross-branch comparison workflow.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

MODE=""
DRY_RUN="true"
while [[ $# -gt 0 ]]; do
    case "$1" in
        --no-dry-run) DRY_RUN="false"; shift ;;
        -h|--help)    MODE="--help"; break ;;
        *)            MODE="$1"; shift ;;
    esac
done
export DRY_RUN

: "${BURST_SIZE:=500}"
: "${VIOLATION_PCT:=50}"
: "${UNIQUE_PCT:=25}"
: "${NAMESPACE:=burst-test}"
: "${IMAGES:=quay.io/centos/centos:7,quay.io/fedora/fedora:38,quay.io/centos/centos:stream9}"
: "${ROX_NAMESPACE:=stackrox}"
: "${METRICS_PORT:=9090}"
: "${LOCAL_PORT:=9090}"
: "${PARALLEL:=50}"

WORK_DIR=$(mktemp -d)

# shellcheck source=lib.sh
source "${SCRIPT_DIR}/lib.sh"

usage() {
    cat <<'EOF'
Usage: ./burst-test.sh <mode> [options]

Modes:
  fast-path    Test spec-only policy evaluation (no image fetching).
               Requires: Privileged Container + Latest tag policies enforced.
  slow-path    Test image coalescing and caching in the enrichment path.
               Runs two phases: cold cache (pods restarted) then warm cache.
               Requires: at least one enrichment-required policy (e.g. Image Age).

Options:
  --no-dry-run Apply deployments for real (replicas=0). Useful for creating
               persistent deployment objects for reprocessor profiling.
  -h, --help   Show this help message.

Environment variables:
  BURST_SIZE       Number of deployments to create           (default: 500)
  VIOLATION_PCT    % that violate policy (fast-path only)    (default: 50)
  UNIQUE_PCT       % of BURST_SIZE used as distinct images   (default: 25)
                   Unique count = BURST_SIZE * UNIQUE_PCT / 100.
  IMAGES_FILE      Path to a file with one image per line.
                   Overrides automatic pool generation.
  NAMESPACE        Namespace for test deployments            (default: burst-test)
  IMAGES           Comma-separated images (fast-path only)   (default: quay.io/centos/centos:7,...)
  ROX_NAMESPACE    StackRox namespace                        (default: stackrox)
  METRICS_PORT     Admission controller metrics port         (default: 9090)
  LOCAL_PORT       Local port for kubectl port-forward       (default: 9090)
  PARALLEL         Max concurrent kubectl creates            (default: 50)

fast-path mode:
  Violating deployments use privileged: true + :latest tags.
  Clean deployments use privileged: false + pinned tags from IMAGES.
  Correctness check: denied == VIOLATION_PCT%, allowed == rest.

slow-path mode:
  Image pool is loaded via generate-image-pool.sh (or IMAGES_FILE).
  UNIQUE_PCT of BURST_SIZE determines how many unique images to generate.
  Phase 1 restarts admission-control pods to flush caches.
  Phase 2 reuses warm caches from phase 1.
  Reports coalesce ratio, cache hit rate, and fetch counts.
  With --no-dry-run, deployments are created with replicas=0.
EOF
    exit 0
}

if [[ "${MODE}" == "-h" || "${MODE}" == "--help" ]]; then
    usage
fi

if [[ "${MODE}" != "fast-path" && "${MODE}" != "slow-path" ]]; then
    echo "ERROR: mode must be 'fast-path' or 'slow-path'" >&2
    echo "Run with -h for usage." >&2
    exit 1
fi

for cmd in kubectl curl bc awk; do
    command -v "$cmd" &>/dev/null || { echo "ERROR: '$cmd' required but not found" >&2; exit 1; }
done

readonly METRICS_NETPOL="admission-control-metrics-allow"
trap cleanup EXIT

# ---------------------------------------------------------------------------
# burst-test-specific helpers
# ---------------------------------------------------------------------------

cleanup() {
    kill_port_forwards
    kubectl -n "$ROX_NAMESPACE" delete networkpolicy "$METRICS_NETPOL" \
        --ignore-not-found &>/dev/null || true
    kubectl delete namespace "$NAMESPACE" --ignore-not-found &>/dev/null || true
    [[ -n "${WORK_DIR:-}" ]] && rm -rf "$WORK_DIR"
}

allow_metrics_ingress() {
    allow_metrics_netpol "$METRICS_NETPOL" "admission-control"
}

restart_admission_control() {
    echo "Restarting admission-control pods..."
    kubectl -n "$ROX_NAMESPACE" rollout restart deployment/admission-control >/dev/null
    kubectl -n "$ROX_NAMESPACE" rollout status deployment/admission-control --timeout=120s >/dev/null

    local attempts=0
    while true; do
        local terminating
        terminating=$(kubectl -n "$ROX_NAMESPACE" get pod -l app=admission-control \
            --no-headers 2>/dev/null | grep -c -v "Running" || true)
        [[ "$terminating" -eq 0 ]] && break
        attempts=$((attempts + 1))
        if [[ "$attempts" -ge 60 ]]; then
            echo "ERROR: old pods still terminating after 60s" >&2
            exit 1
        fi
        sleep 1
    done

    sleep 10
    echo "  pods restarted and ready."
}

scrape_metrics() {
    scrape_component "admission-control" "$LOCAL_PORT" "$1"
}

# Backward-compatible wrapper used by reports.
extract_counter() { extract "$@"; }

# ---------------------------------------------------------------------------
# Fast-path manifest generators
# ---------------------------------------------------------------------------

generate_fast_path_deployment() {
    local name="$1" privileged="$2" outfile="$3"
    {
        cat <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${name}
  namespace: ${NAMESPACE}
  labels:
    burst-test: "true"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ${name}
  template:
    metadata:
      labels:
        app: ${name}
    spec:
      containers:
YAML
        local idx=0
        for img in "${IMAGE_LIST[@]}"; do
            local registry="${img%%:*}"
            local short="${registry##*/}"
            local tag
            if [[ "$privileged" == "true" ]]; then
                tag="latest"
            else
                tag="${img#*:}"
            fi
            cat <<YAML
      - name: c${idx}-${short}
        image: ${registry}:${tag}
        securityContext:
          privileged: ${privileged}
YAML
            idx=$((idx + 1))
        done
    } > "$outfile"
}

generate_fast_path_manifests() {
    for (( i = 0; i < VIOLATING_COUNT; i++ )); do
        generate_fast_path_deployment "burst-v-${i}" "true" "${WORK_DIR}/deploy-${i}.yaml"
    done
    for (( i = 0; i < CLEAN_COUNT; i++ )); do
        local j=$(( i + VIOLATING_COUNT ))
        generate_fast_path_deployment "burst-c-${j}" "false" "${WORK_DIR}/deploy-${j}.yaml"
    done
}

# ---------------------------------------------------------------------------
# Reports
# ---------------------------------------------------------------------------

compute_common_deltas() {
    local before="$1" after="$2"
    local m="rox_admission_control"

    FETCH_D=$(delta \
        "$(extract "$before" "${m}_image_fetch_total")" \
        "$(extract "$after"  "${m}_image_fetch_total")")
    DENIED_D=$(delta \
        "$(extract "$before" "${m}_policyeval_review_total" 'result="denied"')" \
        "$(extract "$after"  "${m}_policyeval_review_total" 'result="denied"')")
    ALLOWED_D=$(delta \
        "$(extract "$before" "${m}_policyeval_review_total" 'result="allowed"')" \
        "$(extract "$after"  "${m}_policyeval_review_total" 'result="allowed"')")
    HIT_D=$(delta \
        "$(extract "$before" "${m}_image_cache_operations_total" 'result="hit"')" \
        "$(extract "$after"  "${m}_image_cache_operations_total" 'result="hit"')")
    MISS_D=$(delta \
        "$(extract "$before" "${m}_image_cache_operations_total" 'result="miss"')" \
        "$(extract "$after"  "${m}_image_cache_operations_total" 'result="miss"')")
    SKIP_D=$(delta \
        "$(extract "$before" "${m}_image_cache_operations_total" 'result="skip"')" \
        "$(extract "$after"  "${m}_image_cache_operations_total" 'result="skip"')")

    local dur_count_d dur_sum_d
    dur_count_d=$(delta \
        "$(extract "$before" "${m}_policyeval_review_duration_seconds_count")" \
        "$(extract "$after"  "${m}_policyeval_review_duration_seconds_count")")
    dur_sum_d=$(delta \
        "$(extract "$before" "${m}_policyeval_review_duration_seconds_sum")" \
        "$(extract "$after"  "${m}_policyeval_review_duration_seconds_sum")")
    DUR_AVG=$(avg_or_na "$dur_sum_d" "$dur_count_d")

    local fpr_count_d fpr_sum_d
    fpr_count_d=$(delta \
        "$(extract "$before" "${m}_image_fetches_per_review_count")" \
        "$(extract "$after"  "${m}_image_fetches_per_review_count")")
    fpr_sum_d=$(delta \
        "$(extract "$before" "${m}_image_fetches_per_review_sum")" \
        "$(extract "$after"  "${m}_image_fetches_per_review_sum")")
    FPR_AVG=$(avg_or_na "$fpr_sum_d" "$fpr_count_d")
}

report_fast_path() {
    local before="$1" after="$2"
    compute_common_deltas "$before" "$after"

    cat <<EOF

==============================
  FAST PATH BURST TEST REPORT
==============================
  Burst size:    ${BURST_SIZE}
  Violation %:   ${VIOLATION_PCT}%
  Expected:      ${VIOLATING_COUNT} denied, ${CLEAN_COUNT} allowed
==============================

EOF
    printf "  %-42s %s\n" "Metric" "Delta"
    printf "  %-42s %s\n" "------" "-----"
    printf "  %-42s %s\n" "image_fetch_total"                "$FETCH_D"
    printf "  %-42s %s\n" "image_fetches_per_review (avg)"   "$FPR_AVG"
    printf "  %-42s %s\n" "policyeval_review_total{denied}"  "$DENIED_D"
    printf "  %-42s %s\n" "policyeval_review_total{allowed}" "$ALLOWED_D"
    printf "  %-42s %s\n" "image_cache_operations{hit}"      "$HIT_D"
    printf "  %-42s %s\n" "image_cache_operations{miss}"     "$MISS_D"
    printf "  %-42s %s\n" "image_cache_operations{skip}"     "$SKIP_D"
    printf "  %-42s %s\n" "review_duration_seconds (avg)"    "${DUR_AVG}s"
    echo ""

    local pass=true
    if [[ "$DENIED_D" -ne "$VIOLATING_COUNT" ]]; then
        echo "  FAIL: expected ${VIOLATING_COUNT} denied, got ${DENIED_D}"
        pass=false
    fi
    if [[ "$ALLOWED_D" -ne "$CLEAN_COUNT" ]]; then
        echo "  FAIL: expected ${CLEAN_COUNT} allowed, got ${ALLOWED_D}"
        pass=false
    fi
    if $pass; then
        echo "  PASS: denied=${DENIED_D} allowed=${ALLOWED_D}"
    fi
    echo ""
}

report_slow_path_phase() {
    local phase_name="$1" before="$2" after="$3"
    compute_common_deltas "$before" "$after"

    local coalesce_saved coalesce_pct cache_total cache_hit_pct
    coalesce_saved=$(echo "${TOTAL_IMAGE_REFS} - ${FETCH_D}" | bc)
    coalesce_pct=$(pct_or_na "$coalesce_saved" "$TOTAL_IMAGE_REFS")
    cache_total=$(echo "${HIT_D} + ${MISS_D} + ${SKIP_D}" | bc)
    cache_hit_pct=$(pct_or_na "$HIT_D" "$cache_total")

    local review_total
    review_total=$(echo "${DENIED_D} + ${ALLOWED_D}" | bc)

    cat <<EOF

  ${phase_name}
  --------------------------------
EOF
    printf "  %-42s %s\n" "total_image_refs"                 "$TOTAL_IMAGE_REFS"
    printf "  %-42s %s\n" "image_fetch_total"                "$FETCH_D"
    printf "  %-42s %s\n" "fetches avoided"                  "$coalesce_saved"
    printf "  %-42s %s\n" "coalesce ratio"                   "${coalesce_pct}%"
    printf "  %-42s %s\n" "image_fetches_per_review (avg)"   "$FPR_AVG"
    printf "  %-42s %s\n" "image_cache_operations{hit}"      "$HIT_D"
    printf "  %-42s %s\n" "image_cache_operations{miss}"     "$MISS_D"
    printf "  %-42s %s\n" "image_cache_operations{skip}"     "$SKIP_D"
    printf "  %-42s %s\n" "cache_hit_rate"                   "${cache_hit_pct}%"
    printf "  %-42s %s\n" "reviews completed"                "$review_total"
    printf "  %-42s %s\n" "review_duration_seconds (avg)"    "${DUR_AVG}s"
    echo ""

    local pass=true
    if [[ "$FETCH_D" -eq 0 ]]; then
        echo "  WARN: image_fetch_total delta is 0. No enrichment-required policy may be active."
        pass=false
    fi
    if [[ "$review_total" -ne "$BURST_SIZE" ]]; then
        echo "  FAIL: expected ${BURST_SIZE} reviews, got ${review_total}"
        pass=false
    fi
    if $pass; then
        echo "  PASS: ${FETCH_D} image fetches for ${TOTAL_IMAGE_REFS} image refs, ${review_total} reviews completed"
    fi
    echo ""
}

# ---------------------------------------------------------------------------
# Main flow
# ---------------------------------------------------------------------------

cat <<EOF
========================================
  Admission Controller Burst Test
========================================
  Mode:            ${MODE}
  BURST_SIZE:      ${BURST_SIZE}
  DRY_RUN:         ${DRY_RUN}
  UNIQUE_PCT:      ${UNIQUE_PCT}%
  Namespace:       ${NAMESPACE}

EOF

setup_namespace
allow_metrics_ingress

if [[ "$MODE" == "fast-path" ]]; then
    IFS=',' read -ra IMAGE_LIST <<< "$IMAGES"
    VIOLATING_COUNT=$(( BURST_SIZE * VIOLATION_PCT / 100 ))
    CLEAN_COUNT=$(( BURST_SIZE - VIOLATING_COUNT ))

    echo "Generating manifests (${VIOLATING_COUNT} violating, ${CLEAN_COUNT} clean)..."
    generate_fast_path_manifests

    : "${SKIP_METRICS:=false}"
    if [[ "$SKIP_METRICS" != "true" ]]; then
        echo "Scraping baseline metrics..."
        scrape_metrics "${WORK_DIR}/before.prom"
        echo "  scraped $(wc -l < "${WORK_DIR}/before.prom") lines"
    fi

    run_burst "Fast path"

    if [[ "$SKIP_METRICS" != "true" ]]; then
        echo "Waiting for metrics to settle..."
        sleep 10

        echo "Scraping post-burst metrics..."
        scrape_metrics "${WORK_DIR}/after.prom"
        echo "  scraped $(wc -l < "${WORK_DIR}/after.prom") lines"

        report_fast_path "${WORK_DIR}/before.prom" "${WORK_DIR}/after.prom"
    else
        echo ""
        echo "=============================="
        echo "  FAST PATH BURST TEST REPORT"
        echo "=============================="
        echo "  Burst size:    ${BURST_SIZE}"
        echo "  Violation %:   ${VIOLATION_PCT}%"
        echo "  Expected:      ${VIOLATING_COUNT} denied, ${CLEAN_COUNT} allowed"
        echo "  (metrics scraping skipped)"
        echo "=============================="
        echo ""
        denied=$(grep -c "^denied" "${WORK_DIR}/results.log" 2>/dev/null || true)
        allowed=$(grep -c "^allowed" "${WORK_DIR}/results.log" 2>/dev/null || true)
        pass=true
        if [[ "$denied" -ne "$VIOLATING_COUNT" ]]; then
            echo "  FAIL: expected ${VIOLATING_COUNT} denied, got ${denied}"
            pass=false
        fi
        if [[ "$allowed" -ne "$CLEAN_COUNT" ]]; then
            echo "  FAIL: expected ${CLEAN_COUNT} allowed, got ${allowed}"
            pass=false
        fi
        if $pass; then
            echo "  PASS: denied=${denied} allowed=${allowed}"
        fi
        echo ""
    fi

elif [[ "$MODE" == "slow-path" ]]; then
    generate_slow_path_manifests
    TOTAL_IMAGE_REFS=$BURST_SIZE

    cat <<EOF

==========================================
  SLOW PATH COALESCING BURST TEST REPORT
==========================================
  Burst size:      ${BURST_SIZE}
  Unique pct:      ${UNIQUE_PCT}%
  Unique images:   ${UNIQUE_COUNT} (of ${POOL_SIZE} in pool)
  Total img refs:  ${TOTAL_IMAGE_REFS}
==========================================
EOF

    # --- Phase 1: Cold cache ---
    restart_admission_control

    echo "Scraping baseline metrics (cold cache)..."
    scrape_metrics "${WORK_DIR}/cold_before.prom"

    run_burst "Phase 1 (cold cache)"

    echo "Waiting for metrics to settle..."
    sleep 10

    echo "Scraping post-burst metrics (cold cache)..."
    scrape_metrics "${WORK_DIR}/cold_after.prom"

    report_slow_path_phase "PHASE 1: COLD CACHE" \
        "${WORK_DIR}/cold_before.prom" "${WORK_DIR}/cold_after.prom"

    # --- Phase 2: Warm cache ---
    echo "Scraping baseline metrics (warm cache)..."
    scrape_metrics "${WORK_DIR}/warm_before.prom"

    run_burst "Phase 2 (warm cache)"

    echo "Waiting for metrics to settle..."
    sleep 10

    echo "Scraping post-burst metrics (warm cache)..."
    scrape_metrics "${WORK_DIR}/warm_after.prom"

    report_slow_path_phase "PHASE 2: WARM CACHE" \
        "${WORK_DIR}/warm_before.prom" "${WORK_DIR}/warm_after.prom"
fi
