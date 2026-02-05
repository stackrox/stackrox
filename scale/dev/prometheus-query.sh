#!/bin/bash
set -x

to=$(($(date +%s%3N)))
from=$(($to - 72000000))
#3600 000

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

#get_time_series_for_metric() {
#  metric=$1
#  from=$2
#  to=$3
#
#  result="$(run_query "$metric" "$from" "$to")"
#
#  time_x="$(echo "$result" | jq .results.A.frames[0].data.values[0][])"
#  values_y="$(echo "$result" | jq .results.A.frames[0].data.values[1][])"
#
#  paste <(echo "$time_x") <(echo "$values_y")
#}

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

for table in process_indicators process_baselines process_baseline_results alerts deployments; do
  metric='rox_central_postgres_table_size{namespace=\"stackrox\",table=\"'$table'\"}'
  result="$(get_time_series_for_metric "$metric" "$from" "$to")"
  echo "$result" > "${output_file_prefix}_${table}.txt"

  metric='rox_central_postgres_table_total_bytes{namespace=\"stackrox\",table=\"'$table'\"}'
  result="$(get_time_series_for_metric "$metric" "$from" "$to")"
  echo "$result" > "${output_file_prefix}_${table}_bytes.txt"
done
