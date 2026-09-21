#!/bin/bash
set -x

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
    -u admin:stackrox \
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

# CPU and memory metrics for Central, Central-DB, and Sensor
for container in central central-db; do
  cpu_metric='rate(container_cpu_usage_seconds_total{namespace=\"stackrox\", container=\"'$container'\"}[1m])'
  mem_metric='container_memory_usage_bytes{namespace=\"stackrox\", container=\"'$container'\"}'
  cpu_result="$(get_time_series_for_metric "$cpu_metric" "$from" "$to")"
  mem_result="$(get_time_series_for_metric "$mem_metric" "$from" "$to")"
  echo "$cpu_result" > "${output_file_prefix}_${container}_cpu.txt"
  echo "$mem_result" > "${output_file_prefix}_${container}_mem.txt"
  echo "cpu_result= $cpu_result"
  echo "mem_result= $mem_result"
done

container=sensor
cpu_metric='rate(container_cpu_usage_seconds_total{namespace=\"stackrox\", container=\"'$container'\"}[1m])'
mem_metric='container_memory_usage_bytes{namespace=\"stackrox\", container=\"'$container'\"}'
cpu_result="$(get_time_series_for_metric "$cpu_metric" "$from" "$to")"
mem_result="$(get_time_series_for_metric "$mem_metric" "$from" "$to")"
echo "$cpu_result" > "${output_file_prefix}_${container}_cpu.txt"
echo "$mem_result" > "${output_file_prefix}_${container}_mem.txt"

# Collector runs as a DaemonSet (one pod per node) with multiple containers:
# collector, compliance, node-inventory, and fact (the file-activity monitor).
# Collect CPU/mem per container, but only if the DaemonSet exists. Container
# names are discovered from the DaemonSet so new ones (e.g. fact) are picked up
# automatically. Metrics are summed across pods to report the cluster-wide total
# per container (collector is spread over every node, like the berserker
# workload); sum() is a no-op for the single-pod components above.
if kubectl -n stackrox get daemonset collector > /dev/null 2>&1; then
  collector_containers="$(kubectl -n stackrox get daemonset collector -o jsonpath='{.spec.template.spec.containers[*].name}')"
  for container in $collector_containers; do
    cpu_metric='sum(rate(container_cpu_usage_seconds_total{namespace=\"stackrox\", container=\"'$container'\"}[1m]))'
    mem_metric='sum(container_memory_usage_bytes{namespace=\"stackrox\", container=\"'$container'\"})'
    cpu_result="$(get_time_series_for_metric "$cpu_metric" "$from" "$to")"
    mem_result="$(get_time_series_for_metric "$mem_metric" "$from" "$to")"
    # Only save a file when the query actually returned data points.
    if [[ -n "${cpu_result//[[:space:]]/}" ]]; then
      echo "$cpu_result" > "${output_file_prefix}_${container}_cpu.txt"
    fi
    if [[ -n "${mem_result//[[:space:]]/}" ]]; then
      echo "$mem_result" > "${output_file_prefix}_${container}_mem.txt"
    fi
  done
else
  echo "Collector DaemonSet not found in namespace stackrox; skipping collector metrics."
fi

# Database table sizes - focus on tables relevant to file activity testing
# File activity events may trigger alerts, so monitor alerts table
# Also monitor deployments as file activity is associated with deployments
for table in alerts deployments; do
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
# - file_access_events_received_total: events received from the fact agent
# - file_activity_buffer_drops:        events dropped due to buffer limits/expiry
# - file_activity_buffer_size:         events currently buffered (gauge)
for metric_name in rox_sensor_file_access_events_received_total rox_sensor_file_activity_buffer_drops rox_sensor_file_activity_buffer_size; do
  metric="${metric_name}"'{namespace=\"stackrox\"}'
  result="$(get_time_series_for_metric "$metric" "$from" "$to")" || true
  if [[ -n "$result" ]]; then
    echo "$result" > "${output_file_prefix}_${metric_name}.txt"
  fi
done
