#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STACKROX_DIR="$DIR/../.."

# Monitoring Grafana basic-auth. Defaults to the chart's committed dev password
# (deploy/charts/monitoring/values.yaml) so this works out of the box against a dev
# deploy; override via MONITORING_USER / MONITORING_PASSWORD. Sourced from the chart
# (single source of truth) rather than hardcoded, so no credential literal lives here.
: "${MONITORING_USER:=admin}"
: "${MONITORING_PASSWORD:=$(awk -F': ' '/^password:/{print $2; exit}' "$STACKROX_DIR/deploy/charts/monitoring/values.yaml")}"

to=$(($(date +%s%3N)))
from=$(($to - 72000000))

run_query() {
  metric=$1
  from=$2
  to=$3

  query='{
      "queries": [{
        "datasource": {
          "type": "prometheus",
          "uid": "PBFA97CFB590B2093"
        },
        "exemplar": true,
        "expr": "'$metric'",
        "instant": false,
        "interval": "",
        "intervalFactor": 1,
        "legendFormat": "{{container}}",
        "refId": "A",
        "requestId": "23763571995A",
        "utcOffsetSec": -25200,
        "datasourceId": 1,
        "intervalMs": 120000,
        "maxDataPoints": 528
      }],
      "from": "'"$from"'",
      "to": "'"$to"'"
  }'

  result_json="$(curl -k -X POST 'https://localhost:48443/api/ds/query?ds_type=prometheus&requestId=Q100' \
    -u "${MONITORING_USER}:${MONITORING_PASSWORD}" \
    -H 'Accept: application/json, text/plain, */*' \
    -H 'Content-Type: application/json' \
    -H 'X-Dashboard-Uid: P0Ulb58nk' \
    -H 'X-Datasource-Uid: PBFA97CFB590B2093' \
    -H 'X-Grafana-Device-Id: 683671fb5c022e9ae19801d53c6f0292' \
    -H 'X-Grafana-Org-Id: 1' \
    -H 'X-Panel-Id: 23763571995' \
    -H 'X-Panel-Plugin-Id: timeseries' \
    -H 'X-Plugin-Id: prometheus' \
    -H 'Origin: https://localhost:48443' \
    -H 'Referer: https://localhost:48443/d/P0Ulb58nk/core-dashboard?orgId=1' \
    -d "$query")"

  echo "$result_json" | jq
}

get_time_series_for_metric() {
  metric=$1
  from=$2
  to=$3

  result="$(run_query "$metric" "$from" "$to")"

  time_x=""
  values_y=""

  frame_count=$(echo "$result" | jq '.results.A.frames | length')

  for i in $(seq 0 $((frame_count - 1))); do
    times=$(echo "$result" | jq ".results.A.frames[$i].data.values[0][]" )
    values=$(echo "$result" | jq ".results.A.frames[$i].data.values[1][]" )

    time_x="$time_x"$'\n'"$times"
    values_y="$values_y"$'\n'"$values"
  done

  paste <(echo "$time_x") <(echo "$values_y") | sort
}

output_file_prefix=$1
# Namespace of the secured-cluster components (sensor, collector, fact, berserker).
# Central/central-db and the Postgres table-size metrics always live in the central
# namespace (stackrox); only the sensor-side components can be elsewhere -- e.g. the
# fake workload deploys its sensor into stackrox1. Defaults to stackrox so berserker
# runs (sensor in stackrox) are unaffected; the fake configs override it.
sensor_ns="${2:-stackrox}"

# Namespace to query for a given component: central/central-db are in the central
# namespace; everything else is a secured-cluster component in sensor_ns.
ns_for_container() {
  case "$1" in
    central|central-db) echo "stackrox" ;;
    *) echo "$sensor_ns" ;;
  esac
}

# CPU and memory metrics for Central, Central-DB, and Sensor.
# Two memory series per container:
#  - _mem.txt          = container_memory_usage_bytes: cgroup total charge. This
#                        includes reclaimable page cache and memory the Go runtime
#                        has freed but not yet returned to the OS, so it overstates
#                        "real" pressure.
#  - _mem_workingset   = container_memory_working_set_bytes (usage minus reclaimable
#                        inactive file cache). This is what Kubernetes uses for
#                        OOM/limit decisions, i.e. the memory that actually can't be
#                        reclaimed under pressure.
for container in central central-db; do
  cpu_metric='rate(container_cpu_usage_seconds_total{namespace=\"stackrox\", container=\"'$container'\"}[1m])'
  mem_metric='container_memory_usage_bytes{namespace=\"stackrox\", container=\"'$container'\"}'
  ws_metric='container_memory_working_set_bytes{namespace=\"stackrox\", container=\"'$container'\"}'
  cpu_result="$(get_time_series_for_metric "$cpu_metric" "$from" "$to")"
  mem_result="$(get_time_series_for_metric "$mem_metric" "$from" "$to")"
  ws_result="$(get_time_series_for_metric "$ws_metric" "$from" "$to")"
  echo "$cpu_result" > "${output_file_prefix}_${container}_cpu.txt"
  echo "$mem_result" > "${output_file_prefix}_${container}_mem.txt"
  echo "$ws_result" > "${output_file_prefix}_${container}_mem_workingset.txt"
  echo "cpu_result= $cpu_result"
  echo "mem_result= $mem_result"
done

container=sensor
cpu_metric='rate(container_cpu_usage_seconds_total{namespace=\"'$sensor_ns'\", container=\"'$container'\"}[1m])'
mem_metric='container_memory_usage_bytes{namespace=\"'$sensor_ns'\", container=\"'$container'\"}'
ws_metric='container_memory_working_set_bytes{namespace=\"'$sensor_ns'\", container=\"'$container'\"}'
cpu_result="$(get_time_series_for_metric "$cpu_metric" "$from" "$to")"
mem_result="$(get_time_series_for_metric "$mem_metric" "$from" "$to")"
ws_result="$(get_time_series_for_metric "$ws_metric" "$from" "$to")"
echo "$cpu_result" > "${output_file_prefix}_${container}_cpu.txt"
echo "$mem_result" > "${output_file_prefix}_${container}_mem.txt"
echo "$ws_result" > "${output_file_prefix}_${container}_mem_workingset.txt"

# Collector runs as a DaemonSet (one pod per node) with multiple containers:
# collector, compliance, node-inventory, and fact (the file-activity monitor).
# Collect CPU/mem per container, but only if the DaemonSet exists. Container
# names are discovered from the DaemonSet so new ones (e.g. fact) are picked up
# automatically. Metrics are summed across pods to report the cluster-wide total
# per container (collector is spread over every node, like the berserker
# workload); sum() is a no-op for the single-pod components above.
if kubectl -n "${sensor_ns}" get daemonset collector > /dev/null 2>&1; then
  collector_containers="$(kubectl -n "${sensor_ns}" get daemonset collector -o jsonpath='{.spec.template.spec.containers[*].name}')"
  for container in $collector_containers; do
    cpu_metric='sum(rate(container_cpu_usage_seconds_total{namespace=\"'$sensor_ns'\", container=\"'$container'\"}[1m]))'
    mem_metric='sum(container_memory_usage_bytes{namespace=\"'$sensor_ns'\", container=\"'$container'\"})'
    ws_metric='sum(container_memory_working_set_bytes{namespace=\"'$sensor_ns'\", container=\"'$container'\"})'
    cpu_result="$(get_time_series_for_metric "$cpu_metric" "$from" "$to")"
    mem_result="$(get_time_series_for_metric "$mem_metric" "$from" "$to")"
    ws_result="$(get_time_series_for_metric "$ws_metric" "$from" "$to")"
    # Only save a file when the query actually returned data points.
    if [[ -n "${cpu_result//[[:space:]]/}" ]]; then
      echo "$cpu_result" > "${output_file_prefix}_${container}_cpu.txt"
    fi
    if [[ -n "${mem_result//[[:space:]]/}" ]]; then
      echo "$mem_result" > "${output_file_prefix}_${container}_mem.txt"
    fi
    if [[ -n "${ws_result//[[:space:]]/}" ]]; then
      echo "$ws_result" > "${output_file_prefix}_${container}_mem_workingset.txt"
    fi
  done
else
  echo "Collector DaemonSet not found in namespace ${sensor_ns}; skipping collector metrics."
fi

# Live Go heap (go_memstats_heap_inuse_bytes) for the Go components. This is
# memory actively held by the Go runtime's heap -- the clearest "real, actively
# used" number, in contrast to container_memory_usage_bytes (cgroup total, which
# also counts reclaimable page cache and freed-but-not-returned pages). Only the
# Go services export it: central-db (Postgres) and collector (C++) do not. These
# come from the app /metrics endpoint (same scrape as rox_central_*/rox_sensor_*).
# The monitoring Prometheus scrapes that endpoint via its `stackrox` endpoints
# job, which sets job=<service name> (central/sensor) but NO pod label -- so an
# earlier pod=~"central-.*" selector matched nothing. Select by job instead; that
# also avoids double-counting against the separate kubernetes-pods job (which does
# carry a pod label). Guarded write: skip if absent.
for container in central sensor; do
  ns="$(ns_for_container "$container")"
  heap_metric='go_memstats_heap_inuse_bytes{namespace=\"'$ns'\",job=\"'$container'\"}'
  heap_result="$(get_time_series_for_metric "$heap_metric" "$from" "$to")" || true
  if [[ -n "${heap_result//[[:space:]]/}" ]]; then
    echo "$heap_result" > "${output_file_prefix}_${container}_heap_inuse.txt"
  fi
done

# Restart and OOM counts per component, including the berserker workload itself
# (an OOM-killed berserker stops emitting file events, which shows up as a lower
# observed vs configured event rate). Both are cumulative counters, so the value
# at the end of the run is the total count.
#  - restarts: kube_pod_container_status_restarts_total (kube-state-metrics)
#  - ooms:     container_oom_events_total (cadvisor OOM-kill counter)
# sum() aggregates across pods for the DaemonSets (collector, fact, berserker)
# and is a no-op for the single-pod components. Guarded writes: a metric source
# that isn't scraped in a given cluster just yields no file (OOMs then still
# surface indirectly as restarts).
for container in central central-db sensor collector fact berserker; do
  ns="$(ns_for_container "$container")"
  restarts_metric='sum(kube_pod_container_status_restarts_total{namespace=\"'$ns'\",container=\"'$container'\"})'
  oom_metric='sum(container_oom_events_total{namespace=\"'$ns'\",container=\"'$container'\"})'
  restarts_result="$(get_time_series_for_metric "$restarts_metric" "$from" "$to")" || true
  oom_result="$(get_time_series_for_metric "$oom_metric" "$from" "$to")" || true
  if [[ -n "${restarts_result//[[:space:]]/}" ]]; then
    echo "$restarts_result" > "${output_file_prefix}_${container}_restarts.txt"
  fi
  if [[ -n "${oom_result//[[:space:]]/}" ]]; then
    echo "$oom_result" > "${output_file_prefix}_${container}_ooms.txt"
  fi
done

# Database table sizes - focus on tables relevant to file activity testing
# File activity events may trigger alerts, so monitor alerts table.
# Also monitor deployments as file activity is associated with deployments, and
# process_indicators to see how many process events land in the DB (a second,
# storage-side view of the process-event volume alongside the sensor counter below).
for table in alerts deployments process_indicators; do
  metric='rox_central_postgres_table_size{namespace=\"stackrox\",table=\"'$table'\"}'
  result="$(get_time_series_for_metric "$metric" "$from" "$to")"
  echo "$result" > "${output_file_prefix}_${table}.txt"

  metric='rox_central_postgres_table_total_bytes{namespace=\"stackrox\",table=\"'$table'\"}'
  result="$(get_time_series_for_metric "$metric" "$from" "$to")"
  echo "$result" > "${output_file_prefix}_${table}_bytes.txt"
done

# File activity specific metrics from sensor (if available).
# File access events flow from the fact container to sensor, which counts them
# and buffers them for process enrichment. These are the actual metric names the
# sensor exposes (see sensor/common/detector/metrics/metrics.go and
# sensor/common/metrics/metrics.go); the earlier names
# (rox_sensor_file_activity_events_total / _events_dropped_total) never existed,
# which is why these output files came out empty.
# - file_access_events_received_total:      events received from the fact agent
# - file_activity_buffer_drops:             events dropped due to buffer limits/expiry
# - file_activity_buffer_size:              events currently buffered (gauge)
# - detector_file_access_queue_dropped_total: file accesses dropped because the
#   sensor detector's queue was full (the "sensor event drops" signal; the counter
#   climbing means the detector can't keep up with the incoming file-access rate --
#   see DetectorFileAccessDroppedCount in sensor/common/detector/metrics/metrics.go).
for metric_name in rox_sensor_file_access_events_received_total rox_sensor_file_activity_buffer_drops rox_sensor_file_activity_buffer_size rox_sensor_detector_file_access_queue_dropped_total; do
  metric="${metric_name}"'{namespace=\"'$sensor_ns'\"}'
  result="$(get_time_series_for_metric "$metric" "$from" "$to")" || true
  if [[ -n "$result" ]]; then
    echo "$result" > "${output_file_prefix}_${metric_name}.txt"
  fi
done

# Event drops in the fact agent itself (the Rust file-activity monitor running as
# the `fact` container in the collector DaemonSet). fact exposes Prometheus metrics
# at :9001/metrics with the registry prefix `stackrox_fact` (see the fact repo:
# fact/src/metrics/exporter.rs Registry::with_prefix, and mod.rs for the counter
# families). Each counter is a Family labeled `label=Added|Dropped|Ignored|...`
# (rendered verbatim, capitalized -- prometheus-client derive stringifies the enum
# variant). We record drops at the three pipeline stages, summed across the per-node
# collector pods (cluster-wide total, like the collector metrics above):
#   - bpf_events        Dropped: kernel ring-buffer -> userspace, fact can't drain fast enough
#   - rate_limiter_events Dropped: shed by FACT_RATE_LIMIT
#   - output_grpc_events  Dropped: couldn't be sent on to sensor
# fact is scraped only via the `kubernetes-pods` Prometheus job (there is no Service
# on port 9001), so these require fact.exposeMonitoring=true; guarded writes skip a
# stage that returns nothing (metric absent / monitoring not exposed).
declare -A fact_drop_metric=(
  [fact_bpf_dropped]='sum(stackrox_fact_bpf_events_total{namespace=\"'$sensor_ns'\",label=\"Dropped\"})'
  [fact_rate_limiter_dropped]='sum(stackrox_fact_rate_limiter_events_total{namespace=\"'$sensor_ns'\",label=\"Dropped\"})'
  [fact_grpc_dropped]='sum(stackrox_fact_output_grpc_events_total{namespace=\"'$sensor_ns'\",label=\"Dropped\"})'
)
for out_name in "${!fact_drop_metric[@]}"; do
  result="$(get_time_series_for_metric "${fact_drop_metric[$out_name]}" "$from" "$to")" || true
  if [[ -n "${result//[[:space:]]/}" ]]; then
    echo "$result" > "${output_file_prefix}_${out_name}.txt"
  fi
done

# Process and network event volume received by Sensor. This tells us how much
# non-file-activity load Sensor is carrying, so it can be compared across modes and
# against the configured rate.
#  - network_flows_received: rox_sensor_total_network_flows_sensor_received_counter,
#    a plain Counter of flows Sensor received from Collector
#    (sensor/common/metrics/metrics.go). sum() collapses per-instance series.
#  - process_enrichment_hits / _drops: rox_sensor_process_enrichment_{hits,drops},
#    the count of process indicators successfully enriched (hits) or dropped (drops)
#    in the process-signal pipeline. This is the pubsub-safe process-throughput
#    signal: the earlier detector queue ADD counter
#    (detector_process_indicator_queue_operations_total) is only incremented on the
#    legacy non-pubsub detector path (sensor/common/detector/detector.go), so it is
#    empty whenever ROX_SENSOR_PUBSUB is enabled (the default). The file-access
#    detector queue counter had the same problem and is dropped;
#    file_access_events_received_total (scraped above) is the file-activity signal.
declare -A received_metric=(
  [network_flows_received]='sum(rox_sensor_total_network_flows_sensor_received_counter{namespace=\"'$sensor_ns'\"})'
  [process_enrichment_hits]='sum(rox_sensor_process_enrichment_hits{namespace=\"'$sensor_ns'\"})'
  [process_enrichment_drops]='sum(rox_sensor_process_enrichment_drops{namespace=\"'$sensor_ns'\"})'
  # Enriched process indicators dropped by the per-container process filter before
  # being sent to Central (the gap between enrichment and Central intake).
  [process_filter_rejected]='sum(rox_sensor_process_filter_rejected_total{namespace=\"'$sensor_ns'\"})'
)
for out_name in "${!received_metric[@]}"; do
  result="$(get_time_series_for_metric "${received_metric[$out_name]}" "$from" "$to")" || true
  if [[ -n "${result//[[:space:]]/}" ]]; then
    echo "$result" > "${output_file_prefix}_${out_name}.txt"
  fi
done

# Alert generation on Central (rox_central_alert_outcome_total, labeled by outcome:
# new / updated / resolved -- central/detection/alertmanager/metrics.go). This is the
# direct "alerts being generated" signal and, being a Central metric, is scraped
# regardless of the sensor pubsub path. Runtime alerts coalesce per (deployment,
# policy): expect a burst of `new` then `updated` climbing as violations fold in.
# Queried in the central namespace (stackrox), not sensor_ns.
declare -A alert_outcome_metric=(
  [alert_outcome_new]='sum(rox_central_alert_outcome_total{namespace=\"stackrox\",outcome=\"new\"})'
  [alert_outcome_updated]='sum(rox_central_alert_outcome_total{namespace=\"stackrox\",outcome=\"updated\"})'
  [alert_outcome_resolved]='sum(rox_central_alert_outcome_total{namespace=\"stackrox\",outcome=\"resolved\"})'
)
for out_name in "${!alert_outcome_metric[@]}"; do
  result="$(get_time_series_for_metric "${alert_outcome_metric[$out_name]}" "$from" "$to")" || true
  if [[ -n "${result//[[:space:]]/}" ]]; then
    echo "$result" > "${output_file_prefix}_${out_name}.txt"
  fi
done

# Central's sharded deduping worker queue (central/sensor/service/connection/
# worker_queue.go: 16 shards + 1, each a DedupingQueue). This is the queue that grew
# unbounded in the file-activity tests when alerts arrived faster than they drained.
#  - *_queue_add / *_queue_remove: rox_central_sensor_event_queue{Operation=Add|Remove,
#    Type=...} -- enqueue vs dequeue ops per message type (Operation is capitalized,
#    from pkg/metrics Op.String()). Add rate = Central's intake rate for that type;
#    Add >> Remove means a growing backlog (the memory-blowup signature).
#  - process_indicator_{deduped,passed}: rox_central_sensor_event_deduper{status,type}
#    -- how many ProcessIndicator messages the pre-queue hash deduper drops vs passes.
#    (Runtime AlertResults skip this deduper, so only ProcessIndicator is meaningful.)
# All central-namespace (stackrox), not sensor_ns. Type values are AlertResults /
# ProcessIndicator (reflectutils.Type, last _-segment; see sensorevents.go).
declare -A central_queue_metric=(
  [alert_queue_add]='sum(rox_central_sensor_event_queue{namespace=\"stackrox\",Operation=\"Add\",Type=\"AlertResults\"})'
  [alert_queue_remove]='sum(rox_central_sensor_event_queue{namespace=\"stackrox\",Operation=\"Remove\",Type=\"AlertResults\"})'
  [process_indicator_queue_add]='sum(rox_central_sensor_event_queue{namespace=\"stackrox\",Operation=\"Add\",Type=\"ProcessIndicator\"})'
  [process_indicator_queue_remove]='sum(rox_central_sensor_event_queue{namespace=\"stackrox\",Operation=\"Remove\",Type=\"ProcessIndicator\"})'
  [process_indicator_deduped]='sum(rox_central_sensor_event_deduper{namespace=\"stackrox\",status=\"deduped\",type=\"ProcessIndicator\"})'
  [process_indicator_passed]='sum(rox_central_sensor_event_deduper{namespace=\"stackrox\",status=\"passed\",type=\"ProcessIndicator\"})'
)
for out_name in "${!central_queue_metric[@]}"; do
  result="$(get_time_series_for_metric "${central_queue_metric[$out_name]}" "$from" "$to")" || true
  if [[ -n "${result//[[:space:]]/}" ]]; then
    echo "$result" > "${output_file_prefix}_${out_name}.txt"
  fi
done
