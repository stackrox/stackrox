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
  sleep $GENERATE_TIMEOUT
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
  sleep $TEST_TIMEOUT
  curl -s $PROMETHEUS_ENDPOINT/api/v1/query?query=$PROMETHEUS_QUERY > "$PROMETHEUS_DUMP" || true
  kill "$SENSOR_PID"
  [[ "$VERBOSE" == "false" ]] || echo "Test done"
}

function exec_option() {
  [[ "$RUN_BUILD" == "yes" ]] && build_local_sensor
  [[ "$RUN_GENERATE" == "yes" ]] && generate_k8s_events
  [[ "$RUN_TEST" == "yes" ]] && run_test
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
