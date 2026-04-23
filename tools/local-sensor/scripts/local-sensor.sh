#!/usr/bin/env bash
set -eou pipefail

LOCAL_SENSOR_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )"/.. && pwd)"
STACKROX_DIR="$(cd "$( dirname "${BASH_SOURCE[0]}" )"/../../.. && pwd)"
OUTPUT_DIR=$LOCAL_SENSOR_DIR/out

K8S_EVENTS_FILE=$OUTPUT_DIR/trace.jsonl
FAKE_WORKLOAD_FILE=$STACKROX_DIR/scale/workloads/default.yaml
POLICIES_FILE=$STACKROX_DIR/sensor/tests/data/policies.json
TIME_FILE=$OUTPUT_DIR/time.txt
LOCAL_SENSOR_BIN=local-sensor
EXEC=$OUTPUT_DIR/$LOCAL_SENSOR_BIN
PROMETHEUS_ENDPOINT=http://localhost:9090
ROX_METRICS_PORT=:9091
PROMETHEUS_QUERY=rox_sensor_sensor_events
PROMETHEUS_DUMP=$OUTPUT_DIR/sensor_events_dump.json

RUN_BUILD="no"
RUN_GENERATE="no"
GENERATE_TIMEOUT=120
RUN_TEST="no"
TEST_TIMEOUT=600
VERBOSE="false"

# File activity load test settings
RUN_FILE_ACTIVITY_LOAD="no"
FILE_ACTIVITY_RATE=100
FILE_ACTIVITY_PATHS=50
FILE_ACTIVITY_DURATION=60
FILE_ACTIVITY_HOSTNAME="fake-collector"
FILE_ACTIVITY_CONTAINER=""
FILE_ACTIVITY_METRICS_DUMP=$OUTPUT_DIR/file_activity_metrics.json
FILE_ACTIVITY_REPORT=$OUTPUT_DIR/file_activity_report.txt

function build_local_sensor() {
  [[ "$VERBOSE" == "false" ]] || echo "Building local-sensor: $LOCAL_SENSOR_BIN"
  mkdir -p "$OUTPUT_DIR"
  go build -o "$EXEC" "$LOCAL_SENSOR_DIR"/main.go > "$OUTPUT_DIR"/build.log 2>&1
  [[ "$VERBOSE" == "false" ]] || echo "Build done"
}

function generate_k8s_events() {
  [[ "$VERBOSE" == "false" ]] || echo "Generating k8s events file: $K8S_EVENTS_FILE"
  $EXEC -record -record-out="$K8S_EVENTS_FILE" -with-fakeworkload="$FAKE_WORKLOAD_FILE" -central-out=/dev/null -no-cpu-prof -no-mem-prof > "$OUTPUT_DIR"/generate.log 2>&1 &
  PID=$!
  [[ "$VERBOSE" == "false" ]] || echo "$LOCAL_SENSOR_BIN PID: $PID"
  sleep "$GENERATE_TIMEOUT"
  kill $PID
  [[ "$VERBOSE" == "false" ]] || echo "Generation done"
}

function run_test() {
  [[ "$VERBOSE" == "false" ]] || echo "Running tests with: $K8S_EVENTS_FILE"
  export ROX_METRICS_PORT=$ROX_METRICS_PORT
  { time $EXEC -replay -replay-in="$K8S_EVENTS_FILE" -delay=0s -with-metrics -with-policies="$POLICIES_FILE" -central-out=/dev/null > "$OUTPUT_DIR"/test.log 2>&1 ; } > "$TIME_FILE" 2>&1 &
  TIME_PID=$!
  SENSOR_PID=$(pgrep -P $TIME_PID)
  [[ "$VERBOSE" == "false" ]] || echo "time PID: $TIME_PID"
  [[ "$VERBOSE" == "false" ]] || echo "$LOCAL_SENSOR_BIN PID: $SENSOR_PID"
  sleep "$TEST_TIMEOUT"
  curl -s "$PROMETHEUS_ENDPOINT/api/v1/query?query=$PROMETHEUS_QUERY" > "$PROMETHEUS_DUMP" || true
  kill "$SENSOR_PID"
  [[ "$VERBOSE" == "false" ]] || echo "Test done"
}

function get_raw_metrics_url() {
  echo "http://localhost${ROX_METRICS_PORT}/metrics"
}

function scrape_file_activity_metrics() {
  local output_file=$1
  local metrics_url
  metrics_url=$(get_raw_metrics_url)
  curl -s "$metrics_url" > "$output_file" 2>/dev/null || true
}

function extract_metric() {
  local metrics_file=$1
  local metric_name=$2
  grep "^${metric_name}" "$metrics_file" 2>/dev/null | grep -v "^#" | head -1 | awk '{print $2}' || true
}

function extract_metric_with_label() {
  local metrics_file=$1
  local metric_name=$2
  local label_match=$3
  grep "^${metric_name}{.*${label_match}" "$metrics_file" 2>/dev/null | grep -v "^#" | awk '{print $2}' || true
}

function format_file_activity_report() {
  local metrics_file=$1
  local report_file=$2
  local duration=$3
  local rate=$4
  local paths=$5

  {
    echo "============================================"
    echo "  File Activity Load Test Report"
    echo "============================================"
    echo ""
    echo "Configuration:"
    echo "  Target rate:    $rate events/sec"
    echo "  Unique paths:   $paths"
    echo "  Duration:       ${duration}s"
    echo "  Hostname:       $FILE_ACTIVITY_HOSTNAME"
    echo "  Container ID:   ${FILE_ACTIVITY_CONTAINER:-<none (node-level)>}"
    echo "  Policies file:  $POLICIES_FILE"
    echo ""
    echo "Pipeline metrics:"

    local received dropped
    received=$(extract_metric "$metrics_file" "rox_sensor_file_access_events_received_total")
    dropped=$(extract_metric "$metrics_file" "rox_sensor_detector_file_access_queue_dropped_total")
    # Convert scientific notation (e.g. 2.748109e+06) to integers
    received=$(printf '%.0f' "${received:-0}" 2>/dev/null || echo "0")
    dropped=$(printf '%.0f' "${dropped:-0}" 2>/dev/null || echo "0")

    echo "  Events received:       ${received:-0}"
    echo "  Events dropped:        ${dropped:-0}"

    if [[ -n "$received" && "$received" != "0" ]]; then
      local actual_rate
      actual_rate=$(echo "scale=1; $received / $duration" | bc 2>/dev/null || echo "N/A")
      echo "  Actual rate:           $actual_rate events/sec"
    fi

    if [[ -n "$dropped" && "$dropped" != "0" && -n "$received" ]]; then
      local drop_pct
      drop_pct=$(echo "scale=2; $dropped * 100 / ($received + $dropped)" | bc 2>/dev/null || echo "N/A")
      echo "  Drop rate:             ${drop_pct}%"
    else
      echo "  Drop rate:             0%"
    fi

    echo ""
    echo "Queue operations:"
    local queue_add queue_remove
    queue_add=$(extract_metric_with_label "$metrics_file" "rox_sensor_detector_file_access_queue_operations_total" 'Operation="Add"')
    queue_remove=$(extract_metric_with_label "$metrics_file" "rox_sensor_detector_file_access_queue_operations_total" 'Operation="Remove"')
    echo "  Added:                 ${queue_add:-0}"
    echo "  Removed:               ${queue_remove:-0}"

    echo ""
    echo "Criteria match duration:"
    local match_count match_sum
    match_count=$(extract_metric "$metrics_file" "rox_sensor_file_access_criteria_match_duration_seconds_count")
    match_sum=$(extract_metric "$metrics_file" "rox_sensor_file_access_criteria_match_duration_seconds_sum")
    echo "  Total matches:         ${match_count:-0}"
    echo "  Total time:            ${match_sum:-0}s"
    if [[ -n "$match_count" && "$match_count" != "0" && -n "$match_sum" ]]; then
      local avg_match
      avg_match=$(echo "scale=6; $match_sum / $match_count" | bc 2>/dev/null || echo "N/A")
      echo "  Avg per event:         ${avg_match}s"
    fi

    echo ""
    echo "============================================"
  } > "$report_file"
}

function run_file_activity_load() {
  [[ "$VERBOSE" == "false" ]] || echo "Running file activity load test: rate=$FILE_ACTIVITY_RATE paths=$FILE_ACTIVITY_PATHS duration=${FILE_ACTIVITY_DURATION}s"
  export ROX_METRICS_PORT=$ROX_METRICS_PORT

  local sensor_args=(
    -file-activity-load
    -file-activity-rate="$FILE_ACTIVITY_RATE"
    -file-activity-paths="$FILE_ACTIVITY_PATHS"
    -file-activity-hostname="$FILE_ACTIVITY_HOSTNAME"
    -with-metrics
    -central-out=/dev/null
    -skip-central-output
    -no-cpu-prof
    -no-mem-prof
  )

  if [[ -n "$FILE_ACTIVITY_CONTAINER" ]]; then
    sensor_args+=(-file-activity-container="$FILE_ACTIVITY_CONTAINER")
  fi

  if [[ -n "$POLICIES_FILE" ]]; then
    sensor_args+=(-with-policies="$POLICIES_FILE")
  fi

  $EXEC "${sensor_args[@]}" > "$OUTPUT_DIR"/file_activity_load.log 2>&1 &
  PID=$!
  [[ "$VERBOSE" == "false" ]] || echo "$LOCAL_SENSOR_BIN PID: $PID"

  sleep "$FILE_ACTIVITY_DURATION"

  local metrics_url
  metrics_url=$(get_raw_metrics_url)
  scrape_file_activity_metrics "$FILE_ACTIVITY_METRICS_DUMP"
  [[ "$VERBOSE" == "false" ]] || echo "Metrics scraped to $FILE_ACTIVITY_METRICS_DUMP"

  kill "$PID"

  format_file_activity_report "$FILE_ACTIVITY_METRICS_DUMP" "$FILE_ACTIVITY_REPORT" \
    "$FILE_ACTIVITY_DURATION" "$FILE_ACTIVITY_RATE" "$FILE_ACTIVITY_PATHS"

  cat "$FILE_ACTIVITY_REPORT"
  [[ "$VERBOSE" == "false" ]] || echo "File activity load test done"
}

function exec_option() {
  [[ "$RUN_BUILD" == "yes" ]] && build_local_sensor
  [[ "$RUN_GENERATE" == "yes" ]] && generate_k8s_events
  [[ "$RUN_TEST" == "yes" ]] && run_test
  [[ "$RUN_FILE_ACTIVITY_LOAD" == "yes" ]] && run_file_activity_load
}

function print_header() {
  echo "============== local-sensor =============="
  echo "RUN_BUILD           = $RUN_BUILD"
  echo "RUN_GENERATE        = $RUN_GENERATE"
  echo "RUN_TEST            = $RUN_TEST"
  echo "VERBOSE             = $VERBOSE"
  echo "OUTPUT_DIR          = $OUTPUT_DIR"
  echo "K8S_EVENTS_FILE     = $K8S_EVENTS_FILE"
  echo "FAKE_WORKLOAD_FILE  = $FAKE_WORKLOAD_FILE"
  echo "POLICIES_FILE       = $POLICIES_FILE"
  echo "TIME_FILE           = $TIME_FILE"
  echo "LOCAL_SENSOR_BIN    = $LOCAL_SENSOR_BIN"
  echo "PROMETHEUS_ENDPOINT = $PROMETHEUS_ENDPOINT"
  echo "PROMETHEUS_QUERY    = $PROMETHEUS_QUERY"
  echo "PROMETHEUS_DUMP     = $PROMETHEUS_DUMP"
  echo "ROX_METRICS_PORT    = $ROX_METRICS_PORT"
  if [[ "$RUN_FILE_ACTIVITY_LOAD" == "yes" ]]; then
    echo "--- File Activity Load ---"
    echo "FILE_ACTIVITY_RATE     = $FILE_ACTIVITY_RATE"
    echo "FILE_ACTIVITY_PATHS    = $FILE_ACTIVITY_PATHS"
    echo "FILE_ACTIVITY_DURATION = $FILE_ACTIVITY_DURATION"
    echo "FILE_ACTIVITY_HOSTNAME = $FILE_ACTIVITY_HOSTNAME"
    echo "FILE_ACTIVITY_CONTAINER= ${FILE_ACTIVITY_CONTAINER:-<none>}"
  fi
  echo "=========================================="
}

function print_help() {
      echo "-b, --build                                builds local-sensor"
      echo "-g, --generate                             generates kubernetes events using fake workloads"
      echo "-t, --test                                 runs the test"
      echo "-v, --verbose                              verbose mode"
      echo "--test-duration [duration]                 duration of the test (in seconds)"
      echo "--generate-duration [duration]             duration of the kubernetes events generation step (in seconds)"
      echo "--with-workload [workload file]            ConfigMap with the fake workload definition"
      echo "--with-k8s-trace [kubernetes events file]  kubernetes events file"
      echo "--with-policies [policies file]            policies file"
      echo "--time-result-name [file name]             name of the file containing the results of the time command"
      echo "--local-sensor-bin [binary name]           local-sensor's binary name"
      echo "--prometheus-endpoint [endpoint]           prometheus endpoint"
      echo "--prometheus-query [query]                 query to be executed to retrieve the metrics dump"
      echo "--prometheus-dump-name [file name]         name of the file containing the metrics dump"
      echo "--metrics-endpoint [endpoint]              metrics endpoint"
      echo ""
      echo "File activity load testing:"
      echo "--file-activity-load                       run file activity load test"
      echo "--file-activity-rate [rate]                target events/sec (default: 100, 0 = burst)"
      echo "--file-activity-paths [count]              number of unique file paths (default: 50)"
      echo "--file-activity-duration [seconds]         duration of the load test (default: 60)"
      echo "--file-activity-hostname [hostname]        hostname for generated events (default: fake-collector)"
      echo "--file-activity-container [id]             container ID (default: empty = node-level events)"
      echo ""
      echo "-h, --help                                 this help"
}

function parse_args() {
  for i in "$@"; do
    case "$i" in
      -h|--help)
        print_help
        exit 0
        ;;
      -b|--build)
        RUN_BUILD="yes"
        shift
        ;;
      -g|--generate)
        RUN_GENERATE="yes"
        shift
        ;;
      -t|--test)
        RUN_TEST="yes"
        shift
        ;;
      -v|--verbose)
        VERBOSE="true"
        shift
        ;;
      --local-sensor-bin)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --local-sensor-bin $arg" && exit 1
        LOCAL_SENSOR_BIN=$arg
        EXEC=$OUTPUT_DIR/$LOCAL_SENSOR_BIN
        shift
        ;;
      --with-workload)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --with-workload $arg" && exit 1
        FAKE_WORKLOAD_FILE=$arg
        shift
        ;;
      --with-k8s-trace)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --with-k8s-trace $arg" && exit 1
        K8S_EVENTS_FILE=$OUTPUT_DIR/$arg
        shift
        ;;
      --generate-duration)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --generate-duration $arg" && exit 1
        GENERATE_TIMEOUT=$arg
        shift
        ;;
      --prometheus-endpoint)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --prometheus-endpoint $arg" && exit 1
        PROMETHEUS_ENDPOINT=$arg
        shift
        ;;
      --metrics-endpoint)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --metrics-endpoint $arg" && exit 1
        ROX_METRICS_PORT=$arg
        shift
        ;;
      --test-duration)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --generate-duration $arg" && exit 1
        TEST_TIMEOUT=$arg
        shift
        ;;
      --with-policies)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --with-policies $arg" && exit 1
        POLICIES_FILE=$arg
        shift
        ;;
      --time-result-name)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --time-result-name $arg" && exit 1
        TIME_FILE=$OUTPUT_DIR/$arg
        shift
        ;;
      --prometheus-query)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --prometheus-query $arg" && exit 1
        PROMETHEUS_QUERY=$arg
        shift
        ;;
      --prometheus-dump-name)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --prometheus-dump-name $arg" && exit 1
        PROMETHEUS_DUMP=$OUTPUT_DIR/$arg
        shift
        ;;
      --file-activity-load)
        RUN_FILE_ACTIVITY_LOAD="yes"
        shift
        ;;
      --file-activity-rate)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --file-activity-rate $arg" && exit 1
        FILE_ACTIVITY_RATE=$arg
        shift
        ;;
      --file-activity-paths)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --file-activity-paths $arg" && exit 1
        FILE_ACTIVITY_PATHS=$arg
        shift
        ;;
      --file-activity-duration)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --file-activity-duration $arg" && exit 1
        FILE_ACTIVITY_DURATION=$arg
        shift
        ;;
      --file-activity-hostname)
        shift
        arg=$1
        [[ ${arg:0:1} == "-" ]] && echo "Invalid argument for --file-activity-hostname $arg" && exit 1
        FILE_ACTIVITY_HOSTNAME=$arg
        shift
        ;;
      --file-activity-container)
        shift
        arg=$1
        FILE_ACTIVITY_CONTAINER=$arg
        shift
        ;;
      -*)
        echo "Unknown argument $i"
        exit 1
        ;;
      *)
        ;;
    esac
  done
  [[ "$VERBOSE" == "false" ]] || print_header
}

function main() {
  parse_args "$@"
  exec_option
}

main "$@"
