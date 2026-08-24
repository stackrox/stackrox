#!/usr/bin/env bash
# lib.sh -- Shared functions for admission controller test scripts.
# Source this file; do not execute directly.
#
# Expected variables (set by caller before sourcing):
#   BURST_SIZE, UNIQUE_PCT, PARALLEL, NAMESPACE, ROX_NAMESPACE,
#   METRICS_PORT, WORK_DIR
#
# Optional variables:
#   IMAGES_FILE  - file with one image ref per line (overrides pool generation)
#   DRY_RUN      - "true" (default) for --dry-run=server, "false" for real applies (replicas=0)
#
# Variables defined here:
#   SCANNABLE_POOL, POOL_SIZE, UNIQUE_COUNT, PF_PIDS (if not already set)

[[ -n "${_LIB_SH_SOURCED:-}" ]] && return 0
readonly _LIB_SH_SOURCED=1

for _lib_cmd in kubectl curl bc awk; do
    command -v "$_lib_cmd" &>/dev/null || {
        echo "ERROR: '${_lib_cmd}' is required but not found in PATH" >&2
        exit 1
    }
done
unset _lib_cmd

SCANNABLE_POOL=()
POOL_SIZE=0

LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# load_image_pool <count>
#   Populates SCANNABLE_POOL and POOL_SIZE.
#   If IMAGES_FILE is set, reads images from that file.
#   Otherwise, calls generate-image-pool.sh with the given count.
#   Updates UNIQUE_COUNT if the pool has fewer images than requested.
load_image_pool() {
    local count="${1:?load_image_pool requires a count argument}"
    if [[ -n "${IMAGES_FILE:-}" ]]; then
        if [[ ! -f "$IMAGES_FILE" ]]; then
            echo "ERROR: IMAGES_FILE='${IMAGES_FILE}' not found" >&2
            exit 1
        fi
        mapfile -t SCANNABLE_POOL < "$IMAGES_FILE"
    else
        mapfile -t SCANNABLE_POOL < <("${LIB_DIR}/generate-image-pool.sh" "$count")
    fi
    POOL_SIZE=${#SCANNABLE_POOL[@]}
    if [[ "$POOL_SIZE" -eq 0 ]]; then
        echo "ERROR: image pool is empty" >&2
        exit 1
    fi
    if [[ "$POOL_SIZE" -lt "$UNIQUE_COUNT" ]]; then
        echo "  WARN: pool has ${POOL_SIZE} images, capping UNIQUE_COUNT from ${UNIQUE_COUNT}." >&2
        UNIQUE_COUNT=$POOL_SIZE
    fi
    echo "  Pool: ${POOL_SIZE} images loaded."
}

: "${PF_PIDS:=}"
if [[ -z "$PF_PIDS" ]]; then
    PF_PIDS=()
fi

# ---------------------------------------------------------------------------
# Infrastructure helpers
# ---------------------------------------------------------------------------

kill_port_forwards() {
    for pid in "${PF_PIDS[@]}"; do
        kill "$pid" 2>/dev/null || true
        wait "$pid" 2>/dev/null || true
    done
    PF_PIDS=()
}

setup_namespace() {
    if kubectl get namespace "$NAMESPACE" &>/dev/null; then
        kubectl delete namespace "$NAMESPACE" --wait &>/dev/null || true
    fi
    kubectl create namespace "$NAMESPACE" >/dev/null
    kubectl label namespace "$NAMESPACE" \
        pod-security.kubernetes.io/enforce=privileged --overwrite >/dev/null
}

# allow_metrics_netpol <netpol-name> <app-label>
allow_metrics_netpol() {
    local name="$1" app_label="$2"
    kubectl -n "$ROX_NAMESPACE" apply -f - <<EOF >/dev/null
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: ${name}
spec:
  podSelector:
    matchLabels:
      app: ${app_label}
  ingress:
  - ports:
    - port: ${METRICS_PORT}
      protocol: TCP
  policyTypes:
  - Ingress
EOF
}

# scrape_component <app-label> <local-port> <outfile>
scrape_component() {
    local label="$1" local_port="$2" outfile="$3"
    kill_port_forwards
    true > "$outfile"

    local pods
    pods=$(kubectl -n "$ROX_NAMESPACE" get pod -l "app=${label}" \
        --field-selector=status.phase=Running \
        -o jsonpath='{.items[*].metadata.name}')
    if [[ -z "$pods" ]]; then
        echo "  ERROR: no running ${label} pods in namespace ${ROX_NAMESPACE}" >&2
        exit 1
    fi

    for pod in $pods; do
        local scraped=false
        for attempt in 1 2 3; do
            kill_port_forwards
            kubectl -n "$ROX_NAMESPACE" port-forward "$pod" \
                "${local_port}:${METRICS_PORT}" >/dev/null 2>&1 &
            PF_PIDS+=($!)
            sleep 2

            for _ in $(seq 1 20); do
                if curl -sf "http://localhost:${local_port}/metrics" >> "$outfile" 2>/dev/null; then
                    scraped=true
                    break
                fi
                sleep 0.5
            done
            kill_port_forwards
            sleep 1

            if $scraped; then
                break
            fi
            echo "  WARN: scrape attempt ${attempt}/3 failed for ${label} pod ${pod}, retrying..." >&2
        done

        if ! $scraped; then
            echo "  ERROR: could not scrape metrics from ${label} pod ${pod} after 3 attempts" >&2
            exit 1
        fi
    done
}

# ---------------------------------------------------------------------------
# Metric math
# ---------------------------------------------------------------------------

# Print n in fixed-point form. awk's default OFMT (%.6g) emits 1.23e+06 for
# values >= 1e6, and bc cannot parse the '+' ("Parse error: bad character '+'").
_awk_fmt='
function fmt(n) {
    x = sprintf("%.6f", n+0)
    sub(/0+$/, "", x)
    sub(/\.$/, "", x)
    if (x == "" || x == "-" || x == "-0") x = "0"
    return x
}
'

# Sum a Prometheus counter/gauge across all lines matching metric + optional labels.
extract() {
    local file="$1" metric="$2" labels="${3:-}"
    if [[ ! -f "$file" ]] || [[ ! -s "$file" ]]; then
        echo "0"
        return
    fi
    if [[ -n "$labels" ]]; then
        { grep "^${metric}[{ ]" "$file" 2>/dev/null | grep "$labels" || true; } \
            | awk "${_awk_fmt}"'{s+=$NF} END{print fmt(s)}'
    else
        { grep "^${metric}[{ ]" "$file" 2>/dev/null || true; } \
            | awk "${_awk_fmt}"'{s+=$NF} END{print fmt(s)}'
    fi
}

delta() {
    awk -v a="$1" -v b="$2" "${_awk_fmt}"'BEGIN{print fmt(b-a)}'
}

avg_or_na() {
    awk -v s="$1" -v c="$2" 'BEGIN{
        if (c+0 > 0) printf "%.4f\n", s/c
        else print "N/A"
    }'
}

pct_or_na() {
    awk -v n="$1" -v d="$2" 'BEGIN{
        if (d+0 > 0) printf "%.1f\n", n*100/d
        else print "N/A"
    }'
}

row() { printf "  %-50s %s\n" "$1" "$2"; }

# ---------------------------------------------------------------------------
# Slow-path manifest generation
# ---------------------------------------------------------------------------

generate_slow_path_deployment() {
    local name="$1" outfile="$2"
    shift 2
    local replicas=1
    if [[ "${DRY_RUN:-true}" == "false" ]]; then
        replicas=0
    fi
    {
        cat <<YAML
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${name}
  namespace: ${NAMESPACE}
  labels:
    burst-test: "true"
YAML
        if [[ "${DRY_RUN:-true}" == "false" ]]; then
            cat <<YAML
  annotations:
    admission.stackrox.io/break-glass: "profile"
YAML
        fi
        cat <<YAML
spec:
  replicas: ${replicas}
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
        for img in "$@"; do
            cat <<YAML
      - name: c${idx}
        image: ${img}
YAML
            idx=$((idx + 1))
        done
    } > "$outfile"
}

# write_slow_path_manifests writes BURST_SIZE deployments from SCANNABLE_POOL.
# Requires: BURST_SIZE, UNIQUE_COUNT, SCANNABLE_POOL, WORK_DIR, NAMESPACE
write_slow_path_manifests() {
    for (( i = 0; i < BURST_SIZE; i++ )); do
        generate_slow_path_deployment "burst-${i}" \
            "${WORK_DIR}/deploy-${i}.yaml" \
            "${SCANNABLE_POOL[$(( i % UNIQUE_COUNT ))]}"
    done
    echo "  Generated ${BURST_SIZE} manifests (${UNIQUE_COUNT} unique images)."
}

generate_slow_path_manifests() {
    UNIQUE_COUNT=$(( BURST_SIZE * UNIQUE_PCT / 100 ))
    [[ "$UNIQUE_COUNT" -lt 1 ]] && UNIQUE_COUNT=1

    load_image_pool "$UNIQUE_COUNT"
    write_slow_path_manifests
}

# ---------------------------------------------------------------------------
# Reassess trigger + wait
# ---------------------------------------------------------------------------

# trigger_and_wait_for_reprocessor
#   Requires: ROX_NAMESPACE, ROX_ADMIN_USER, ROX_PASSWORD,
#             ROX_CENTRAL_ADDRESS, REASSESS_WAIT_TIMEOUT, WORK_DIR
trigger_and_wait_for_reprocessor() {
    local central_pod
    central_pod=$(kubectl -n "$ROX_NAMESPACE" get pod -l app=central \
        --field-selector=status.phase=Running \
        -o jsonpath='{.items[0].metadata.name}')
    if [[ -z "$central_pod" ]]; then
        echo "  ERROR: no running Central pod found." >&2
        exit 1
    fi

    local log_marker="Done sending reprocess deployments messages"

    echo "  Starting Central log tail on ${central_pod}..."
    kubectl -n "$ROX_NAMESPACE" logs -f "$central_pod" --since=10s \
        > "${WORK_DIR}/central_logs.txt" 2>/dev/null &
    local log_pid=$!
    sleep 1

    echo "  Triggering reassess via Central API..."
    local http_code
    http_code=$(curl -sk -o /dev/null -w '%{http_code}' \
        -u "${ROX_ADMIN_USER}:${ROX_PASSWORD}" \
        -X POST "https://${ROX_CENTRAL_ADDRESS}/v1/policies/reassess" || true)
    if [[ "$http_code" != "200" ]]; then
        kill "$log_pid" 2>/dev/null || true
        echo "  ERROR: reassess returned HTTP ${http_code} (curl may have failed)" >&2
        exit 1
    fi
    echo "  Reassess triggered (HTTP 200)."

    echo "  Waiting for reprocessor to complete (timeout ${REASSESS_WAIT_TIMEOUT}s)..."
    echo "  Watching for: \"${log_marker}\""
    local start_ts
    start_ts=$(date +%s)

    while true; do
        local elapsed=$(( $(date +%s) - start_ts ))
        if [[ "$elapsed" -ge "$REASSESS_WAIT_TIMEOUT" ]]; then
            echo ""
            echo "  ERROR: timeout after ${REASSESS_WAIT_TIMEOUT}s waiting for reprocessor." >&2
            kill "$log_pid" 2>/dev/null || true
            exit 1
        fi
        if grep -q "$log_marker" "${WORK_DIR}/central_logs.txt" 2>/dev/null; then
            echo ""
            echo "  Detected log marker (elapsed ${elapsed}s)."
            local summary
            summary=$(grep "Successfully reprocessed" "${WORK_DIR}/central_logs.txt" | tail -1)
            [[ -n "$summary" ]] && echo "  Central: ${summary##*: }"
            break
        fi
        sleep 5
        printf "\r  Watching logs... %ds elapsed" "$elapsed"
    done

    kill "$log_pid" 2>/dev/null || true
    wait "$log_pid" 2>/dev/null || true
}

# ---------------------------------------------------------------------------
# Pre-scan helpers (warm Central DB via REST API)
# ---------------------------------------------------------------------------

# prescan_image <image-ref>
#   Triggers a ScanImage request to Central to populate scan data in the DB.
#   Requires: ROX_CENTRAL_ADDRESS, ROX_ADMIN_USER, ROX_PASSWORD
# Prints the HTTP status code on stdout. Returns 0 only for HTTP 200.
prescan_image() {
    local image="$1"
    local http_code
    http_code=$(curl -sk -o /dev/null -w '%{http_code}' \
        -u "${ROX_ADMIN_USER}:${ROX_PASSWORD}" \
        --max-time 120 \
        -X POST "https://${ROX_CENTRAL_ADDRESS}/v1/images/scan" \
        -H "Content-Type: application/json" \
        -d "{\"image_name\": \"${image}\", \"force\": true}")
    echo "$http_code"
    [[ "$http_code" == "200" ]]
}

# prescan_pool <parallel>
#   Pre-scans SCANNABLE_POOL[0..UNIQUE_COUNT-1] via Central's ScanImage API.
#   Requires: ROX_CENTRAL_ADDRESS, ROX_ADMIN_USER, ROX_PASSWORD,
#             SCANNABLE_POOL, UNIQUE_COUNT
prescan_pool() {
    local parallel="${1:-5}"
    local completed=0 failed=0
    local pids=()

    for (( i = 0; i < UNIQUE_COUNT; i++ )); do
        local img="${SCANNABLE_POOL[$i]}"
        (
            if code=$(prescan_image "$img"); then
                echo "ok ${img}"
            else
                echo "fail ${img} http=${code}"
            fi
        ) >> "${WORK_DIR}/prescan.log" &
        pids+=($!)

        if [[ "${#pids[@]}" -ge "$parallel" ]]; then
            wait "${pids[@]}" 2>/dev/null || true
            pids=()
        fi
    done
    if [[ "${#pids[@]}" -gt 0 ]]; then
        wait "${pids[@]}" 2>/dev/null || true
    fi

    completed=$(grep -c "^ok " "${WORK_DIR}/prescan.log" 2>/dev/null || true)
    failed=$(grep -c "^fail " "${WORK_DIR}/prescan.log" 2>/dev/null || true)
    echo "  Pre-scan complete: ${completed} succeeded, ${failed} failed (of ${UNIQUE_COUNT})."

    if [[ "$failed" -gt 0 ]]; then
        echo "  Failed images:" >&2
        grep "^fail " "${WORK_DIR}/prescan.log" | while read -r _ rest; do
            echo "    ${rest}" >&2
        done
    fi
}

# filter_pool_to_prescan_success keeps only images that returned HTTP 200.
# Sets SCANNABLE_POOL, UNIQUE_COUNT, BURST_SIZE, and POOL_SIZE to that set so
# the burst never includes tags that were not stored in Central's DB.
filter_pool_to_prescan_success() {
    local ok_file="${WORK_DIR}/prescan.ok"
    grep "^ok " "${WORK_DIR}/prescan.log" 2>/dev/null | awk '{print $2}' > "$ok_file" || true
    mapfile -t SCANNABLE_POOL < "$ok_file"
    UNIQUE_COUNT=${#SCANNABLE_POOL[@]}
    if [[ "$UNIQUE_COUNT" -eq 0 ]]; then
        echo "ERROR: every prescan failed. Check ROX_PASSWORD, ROX_CENTRAL_ADDRESS, and Scanner health." >&2
        echo "  Sample failures:" >&2
        grep "^fail " "${WORK_DIR}/prescan.log" | head -5 >&2 || true
        exit 1
    fi
    BURST_SIZE=$UNIQUE_COUNT
    POOL_SIZE=$UNIQUE_COUNT
    echo "  Bursting ${BURST_SIZE} unique pre-scanned images (failed prescans dropped)."
}

# ---------------------------------------------------------------------------
# Burst runner
# ---------------------------------------------------------------------------

run_burst() {
    local label="$1"
    local dry_run_flag=""
    if [[ "${DRY_RUN:-true}" == "true" ]]; then
        dry_run_flag="--dry-run=server"
    fi
    echo "${label}: Applying ${BURST_SIZE} deployments (${dry_run_flag:-live}), parallelism=${PARALLEL}..."
    true > "${WORK_DIR}/results.log"

    local pids=()
    for f in "${WORK_DIR}"/deploy-*.yaml; do
        (
            # shellcheck disable=SC2086
            if kubectl create ${dry_run_flag} -f "$f" 2>/dev/null; then
                echo "allowed"
            else
                echo "denied"
            fi
        ) >> "${WORK_DIR}/results.log" &
        pids+=($!)
        if [[ "${#pids[@]}" -ge "$PARALLEL" ]]; then
            wait "${pids[@]}" 2>/dev/null || true
            pids=()
        fi
    done
    if [[ "${#pids[@]}" -gt 0 ]]; then
        wait "${pids[@]}" 2>/dev/null || true
    fi

    local denied allowed
    denied=$(grep -c "^denied" "${WORK_DIR}/results.log" 2>/dev/null || true)
    allowed=$(grep -c "^allowed" "${WORK_DIR}/results.log" 2>/dev/null || true)
    echo "  Results: ${denied} denied, ${allowed} allowed"
}
