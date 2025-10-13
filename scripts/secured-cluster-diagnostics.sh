#!/usr/bin/env bash
set -eu

KUBE_COMMAND=${KUBE_COMMAND:-kubectl}

usage() {
    echo "$0 <namespace> <output-dir> <profile-secs>"
    echo "e.g. $0 stackrox /logs 30"
}

die() {
    echo >&2 "$@"
    exit 1
}

function get_url {
    local url=$1
    if [ -x "$(which curl)" ]; then
        curl -sfL "${url}"
    elif [ -x "$(which wget)" ] ; then
        wget -q -O - "${url}"
    else
        die "missing wget and curl, please install one"
    fi
}

function post_url {
    local url=$1
    local post_data=$2
    if [ -x "$(which curl)" ]; then
        curl -sfkL -X POST -d "$post_data" "$url"
    elif [ -x "$(which wget)" ] ; then
        wget -q --post-data "$post_data" "$url"
    else
        die "missing wget and curl, please install one"
    fi
}

shutdown_port_forward() {
    local PID="$1"
    rm -f "/tmp/port-forward.$$"
    kill "${PID}"
}

# Local port forward to given pod and port, prints child PID
create_port_forward() {
    local pod="$1"
    local ns="$2"
    local port="$3"
    local out="/tmp/port-forward.$$"
    "${KUBE_COMMAND}" -n "$ns" port-forward "$pod" "$port" &> "$out" &
    PID=$!
    # wait until port-forward is successful
    until grep -q -i "Forwarding from" "$out";
    do
        sleep 1
    done
    echo "$PID"
}

save_prom_metrics() {
    local pod="$1"
    local ns="$2"
    local output_dir="$3"
    local metrics_port="9090"
    local metrics_endp="http://localhost:${metrics_port}"
    echo "${pod}: downloading prometheus metrics"
    PID="$(create_port_forward "$pod" "$ns" "$metrics_port")"
    get_url "${metrics_endp}/metrics" > "${output_dir}/${pod}.txt"
    shutdown_port_forward "${PID}"
}

save_container_log() {
    local pod="$1"
    local ns="$2"
    local container="$3"
    local output_dir="$4"
    echo "${pod}: downloading logs for $container container"
    "${KUBE_COMMAND}" -n "$ns" logs "${pod}" -c "${container}" > "${output_dir}/${pod}-${container}.log"
}

save_pod_yaml() {
    local pod="$1"
    local ns="$2"
    local output_dir="$3"
    echo "${pod}: downloading pod yaml description"
    "${KUBE_COMMAND}" -n "$ns" get pod "${pod}" -o yaml > "${output_dir}/${pod}.yaml"
}

# Query collector profile endpoint and check if key equals value
# e.g., check_profile "supports_heap" "true", returns 0 if supports_heap=true
check_profile() {
    local key="$1"
    local expected="$2"
    local url="http://localhost:8080/profile"

    local val
    val="$(get_url "$url" | grep "\"${key}\"" | awk -F":|," '{print $2}' | tr -d '\"[:space:]')"
    if [[ "$val" != "$expected" ]]; then
        return 1
    fi
    return 0
}

# Trigger or save profile data for cpu and heap from collector pod
collector_pod_profile() {
    local pod="$1"
    local ns="$2"
    local output_dir="$3"
    local type="$4"
    local action="$5"

    local port=8080
    local url="http://localhost:${port}/profile/${type}"

    PID="$(create_port_forward "$pod" "$ns" "$port")"

    if ! check_profile "supports_${type}" "true" ; then
        echo "${pod}: ${type} ${action} not supported"
        shutdown_port_forward "${PID}"
        return 0
    fi

    case $action in
    "on")
        if check_profile "${type}" "on" ; then
            echo "${pod}: ${type} profiling already enabled, stopping..."

            post_url "${url}" "off"
            if check_profile "${type}" "on" ; then
                shutdown_port_forward "${PID}"
                die "${pod}: failed to disable ${type} profiling"
            fi
        fi
        echo "${pod}: starting ${type} profile"
        post_url "${url}" "on"
    ;;
    "off")
        if check_profile "${type}" "empty" ; then
            shutdown_port_forward "${PID}"
            die "${pod}: ${type} profiler in unexpected state"
        fi

        echo "${pod}: stopping ${type} profile"
        post_url "${url}" "off"

        if ! check_profile "${type}" "off" ; then
            shutdown_port_forward "${PID}"
            die "${pod}: failed to disable ${type} profiling"
        fi

        echo "${pod}: downloading ${type} profile"
        get_url "${url}" > "${output_dir}/${pod}-${type}.prof"
        echo "${pod}: clearing ${type} profile"
        post_url "${url}" "empty"
    ;;
    esac
    shutdown_port_forward "${PID}"
}

save_sensor_diagnostics() {
    local ns="$1"
    local output_dir="$2"
    local port="6060"
    local endp="http://localhost:${port}"

    while true; do
        pod="$(${KUBE_COMMAND} get pod -n "$ns" --selector 'app=sensor' --field-selector 'status.phase=Running' --output 'jsonpath={.items..metadata.name}' 2>/dev/null)"
        [ -z  "$pod" ] || break
        sleep 1
    done

    save_pod_yaml "${pod}" "${ns}" "${output_dir}"
    save_container_log "${pod}" "${ns}" "sensor" "${output_dir}"
    save_prom_metrics "${pod}" "${ns}" "${output_dir}"

    PID="$(create_port_forward "$pod" "$ns" "$port")"

    echo "${pod}: downloading heap profile"
    get_url "${endp}/debug/heap" > "${output_dir}/${pod}-heap.pb.gz"
    echo "${pod}: downloading cpu profile (30 sec)"
    get_url "${endp}/debug/pprof/profile" > "${output_dir}/${pod}-cpu.pb.gz"
    shutdown_port_forward "${PID}"
}

save_collector_diagnostics() {
    local ns="$1"
    local output_dir="$2"
    local profile_secs="$3"

    local pods
    while true; do
        read -ra pods <<< \
            "$(${KUBE_COMMAND} get pod -n "$ns" --selector 'app=collector' --field-selector 'status.phase=Running' --output 'jsonpath={.items..metadata.name}' 2>/dev/null)"
        [ -z "${pods[*]}" ] || break
        sleep 1
    done
    echo "Found ${#pods[@]} collector pods: ${pods[*]}"

    if [[ -z "${pods[*]}" ]]; then
        echo "No pods found for collector service"
        exit 0
    fi

    for pod in "${pods[@]}"; do
        save_pod_yaml "${pod}" "${ns}" "${output_dir}"
        save_prom_metrics "${pod}" "${ns}" "${output_dir}"
    done

    # enable profiling in all pods, sleep, then pull profiles
    local profile_types=("heap" "cpu")
    for ptype in "${profile_types[@]}" ; do
        for pod in "${pods[@]}"; do
            collector_pod_profile "${pod}" "${ns}" "${output_dir}" "${ptype}" "on"
        done
        sleep "${profile_secs}"
        for pod in "${pods[@]}"; do
            collector_pod_profile "${pod}" "${ns}" "${output_dir}" "${ptype}" "off"
        done
    done

    # get logs after all actions
    for pod in "${pods[@]}"; do
        save_container_log "$pod" "$ns" "collector" "${output_dir}"
        save_container_log "$pod" "$ns" "compliance" "${output_dir}"
    done
}

main() {
    if [ $# -gt 0 ]; then
        namespace="$1"
    else
        namespace="stackrox"
    fi

    if [ $# -gt 1 ]; then
        output_dir="$2"
    else
        output_dir="/tmp/k8s-service-logs/$namespace/metrics"
    fi
    mkdir -p "${output_dir}"

    if [ $# -gt 2 ]; then
        profile_secs="$3"
    else
        profile_secs="30"
    fi

    save_sensor_diagnostics "$namespace" "$output_dir"
    save_collector_diagnostics "$namespace" "$output_dir" "$profile_secs"
}

main "$@"
