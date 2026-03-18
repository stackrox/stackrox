#!/usr/bin/env bash
# Monitor sensor metrics during secret OOM reproduction.
# Tracks the ResolveAllDeployments() amplification waterfall:
#   resolver queue saturation -> handler blocking -> memory growth
#
# Run this in a separate terminal while local-sensor is running.
#
# Usage: ./secret-oom-repro-monitor.sh [interval_seconds]
#
# Limitations:
# - Parses Prometheus text exposition format with grep+awk. This is
#   fragile if metric names, label sets, or exposition formatting change.
# - Does not use `set -e` because curl failures are expected while the
#   sensor is starting up; they are handled with an explicit check.
# - Does not use `set -o pipefail` because grep exits 1 when a metric
#   line is absent, which would falsely propagate as a pipeline error.

set -u

INTERVAL="${1:-5}"
METRICS_URL="http://localhost:9090/metrics"

echo "Monitoring sensor metrics every ${INTERVAL}s..."
echo "Watching for ResolveAllDeployments() amplification waterfall."
echo "Press Ctrl+C to stop."
echo ""
printf "%-9s  %-9s  %-9s  %-9s  %-9s  %-9s  %-9s  %-12s  %-12s\n" \
    "TIME" "HEAP_MB" "RSS_MB" "RSLVR_Q" "PUBSUB_Q" "OUT_Q" "PODS" "SEC_SYNC" "DEP_SENT"
printf "%s\n" \
    "---------  ---------  ---------  ---------  ---------  ---------  ---------  ------------  ------------"

while true; do
    METRICS=$(curl -s "$METRICS_URL" 2>/dev/null)
    if [ $? -ne 0 ]; then
        echo "$(date +%H:%M:%S)  -- metrics endpoint not available --"
        sleep "$INTERVAL"
        continue
    fi

    HEAP=$(echo "$METRICS" | grep '^go_memstats_alloc_bytes ' | awk '{printf "%.0f", $2/1048576}')
    RSS=$(echo "$METRICS" | grep '^process_resident_memory_bytes ' | awk '{printf "%.0f", $2/1048576}')
    RESOLVER=$(echo "$METRICS" | grep '^rox_sensor_resolver_channel_size ' | awk '{print $2}')
    # PubSub lane queue: the replacement for resolver_channel_size when ROX_SENSOR_PUBSUB=true
    PUBSUB_Q=$(echo "$METRICS" | grep '^rox_sensor_pubsub_lane_queue_size_current{lane_id="KubernetesDispatcherEvent"}' | awk '{printf "%.0f", $2}')
    OUTPUT=$(echo "$METRICS" | grep '^rox_sensor_output_channel_size ' | awk '{print $2}')
    PODS=$(echo "$METRICS" | grep '^rox_sensor_num_pods_in_store{' | awk '{sum+=$2} END {printf "%.0f", sum}')

    # Count synced docker config secrets (ImageIntegration events from Secret dispatcher)
    SEC_SYNC=$(echo "$METRICS" | grep 'rox_sensor_k8s_event_ingestion_to_send_duration_count.*Secret.*ImageIntegration.*total' | awk '{sum+=$2} END {printf "%.0f", sum}')

    # Count deployment events sent due to secrets (the amplified output)
    DEP_SENT=$(echo "$METRICS" | grep 'rox_sensor_k8s_event_ingestion_to_send_duration_count.*Secret.*Deployment.*total' | awk '{sum+=$2} END {printf "%.0f", sum}')

    printf "%-9s  %-9s  %-9s  %-9s  %-9s  %-9s  %-9s  %-12s  %-12s\n" \
        "$(date +%H:%M:%S)" \
        "${HEAP:-?}" \
        "${RSS:-?}" \
        "${RESOLVER:-?}" \
        "${PUBSUB_Q:-?}" \
        "${OUTPUT:-?}" \
        "${PODS:-?}" \
        "${SEC_SYNC:-0}" \
        "${DEP_SENT:-0}"

    sleep "$INTERVAL"
done
