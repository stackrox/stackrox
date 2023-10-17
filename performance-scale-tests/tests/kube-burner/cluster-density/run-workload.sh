#!/usr/bin/env bash

# This script runs the workload in the current directory.

set -eou pipefail
shopt -s inherit_errexit

function log() {
    echo "${1:-}" >&2
}

function log_exit() {
    log "${1:-}"

    exit 1
}

function usage() {
    log "
Usage:
 run-workload.sh [OPTION]

OPTION:
  --num-namespaces     Number of namespaces with defined workload. (default: 100)
  --num-deployments    Number of deployments per namespace. (default: 5)
  --num-pods           Number of pods per deployment. (default: 2)
  --num-configs        Number of secrets/configMaps per deployment. (default 4)
  --num-iterations     Number of time the workload will be executed. (default 1)
  --num-patches        Number of times the deployments will be patched. (default 100)
                       This is only applied if the used template has a job type patch.
  --template           Indicates the template to the config to be used by kube-burner. (default cluster-density-template.yml)
  --kube-burner-path   Path to kube-burner executable. (default: kube-burner)
  --help               Prints help information.

Example:
  run-workload.sh --num-namespaces 20
  run-workload.sh --num-namespaces 2000 --num-deployments 4 --num-pods 1
"
}

function usage_exit() {
    usage

    exit 1
}

function check_command() {
    local cmd="${1:-}"

    echo "- Looking for '${cmd}'"
    command -v "${cmd}" || log_exit "-- Command '${cmd}' required."
    echo "- Found '${cmd}'!"
}

function check_dependencies() {
    echo "--- Checking command dependencies"

    check_command oc

    echo "--- Done!"
}

function run_workload() {
    # We have to export following variables for "envsubst"
    export num_namespaces="${1:-100}"
    export num_deployments="${2:-5}"
    export num_pods="${3:-2}"
    export num_iterations="${4:-1}"
    export num_configs="${5:-4}"
    export num_patches="${6:-100}"
    export resource_name="cluster-density"
    template="${7}"

    local kube_burner_path="${8:-kube-burner}"

    local script_dir
    script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

    echo "Creating workload with following values:"
    echo "Template: ${template}"
    echo "Iterations: ${num_iterations}"
    echo "Namespaces: ${num_namespaces}"
    echo "Deployments per namespace: ${num_deployments}"
    echo "Pods per deployment: ${num_pods}"
    echo "Secrets and Configmaps per deployment: ${num_configs}"

    local prometheus_url
    prometheus_url="https://$(oc get route --namespace openshift-monitoring prometheus-k8s --output jsonpath='{.spec.host}' | xargs)"

    local prometheus_token
    prometheus_token="$(oc serviceaccounts new-token --namespace openshift-monitoring prometheus-k8s)"

    local num_nodes
    num_nodes="$(oc get nodes | wc -l)"

    # Remove header + 3 master nodes
    num_nodes=$(( num_nodes - 4 ))

    # Get node instance type
    local node_type
    node_type="$(oc get machines.machine.openshift.io --namespace openshift-machine-api | grep worker | tail -n 1 | awk '{print $3}')"

    local run_uuid="node-${num_nodes}--${node_type}--dep-${num_deployments}--pod-${num_pods}--workload-${num_namespaces}--run-0"

    echo "--- Starting kube-burner"
    "${kube_burner_path}" init \
        --uuid="${run_uuid}" \
        --config="${script_dir}/${template}" \
        --metrics-profile="${script_dir}/metrics.yml" \
        --alert-profile="${script_dir}/alerts.yml" \
        --skip-tls-verify \
        --timeout=2h \
        --prometheus-url="${prometheus_url}" \
        --token="${prometheus_token}"
    echo "--- Done!"

    echo "Move results to: ${run_uuid}.tar.gz"
    mv collected-metrics.tar.gz "${run_uuid}.tar.gz"
}

function main() {
    local num_namespaces="100"
    local num_deployments="5"
    local num_pods="2"
    local num_iterations="1"
    local num_configs="4"
    local num_patches="100"
    local template="cluster-density-template.yml"
    local kube_burner_path="kube-burner"

    while [[ -n "${1:-}" ]]; do
        case "${1}" in
        "--num-iterations")
            num_iterations="${2:-}"
            shift
            ;;
        "--num-namespaces")
            num_namespaces="${2:-}"
            shift
            ;;
        "--num-deployments")
            num_deployments="${2:-}"
            shift
            ;;
        "--num-pods")
            num_pods="${2:-}"
            shift
            ;;
        "--num-configs")
            num_configs="${2:-}"
            shift
            ;;
        "--num-patches")
            num_patches="${2:-}"
            shift
            ;;
        "--template")
            template="${2:-}"
            shift
            ;;
        "--kube-burner-path")
            kube_burner_path="${2:-}"
            shift
            ;;
        "--help")
            usage_exit
            ;;
        *)
            log "Error: Unknown parameter: ${1:-}"
            usage_exit
            ;;
        esac

        if ! shift; then
            log "Error: Missing parameter argument."
            usage_exit
        fi
    done

    check_dependencies

    echo "--- Checking kube-burner command"
    check_command "${kube_burner_path}"
    echo "--- Done!"

    run_workload "${num_namespaces}" \
      "${num_deployments}" \
      "${num_pods}" \
      "${num_iterations}" \
      "${num_configs}" \
      "${num_patches}" \
      "${template}" \
      "${kube_burner_path}"
}

main "$@"
