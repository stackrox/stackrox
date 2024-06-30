#!/usr/bin/env bash

# This script expect that you have an OpenShift4 cluster avilable 
# and has `kube-burner` installed. 

###################
## Prerequisites ##
###################

KB=$(which kube-burner)
if [ "$?" != 0 ]; then
    echo "kube-burner binary not found. Make sure to install it and have it available in \$PATH"
    echo "https://github.com/cloud-bulldozer/kube-burner/releases"
    exit
fi

OC=$(which oc)
if [ "$?" != 0 ]; then
    echo "oc command not found"
    exit
fi

# Make sure there is an OpenShift cluster available
results=$($OC get projects | grep "openshift")
if [ "$?" != 0 ]; then
    echo "Failed to run 'oc get projects'. Make sure the available cluster is an OpenShift cluster"
fi

if [ -z "$results" ]; then
    echo "'oc get projects' returns not openshift-* projects. Make sure the available cluster is an OpenShift cluster"
fi

function central_deployment_count() {
    $OC -n stackrox get deployments -l app=central -o go-template='{{printf "%d\n" (len  .items)}}'
}

##################
## Run Scneario ##
##################

function run_scenario() {
    local image_tag="$1"
    local iterations="$2"
    local output="$3"

    echo "Running scenario for image tag: $image_tag"

    MAIN_IMAGE_TAG="$image_tag" ./deploy/openshift/deploy.sh

    # Run
    $KB ocp cluster-density-v2 --churn=false --local-indexing=true --iterations="$iterations" --timeout=30m

    $OC -n stackrox port-forward deploy/sensor 6060:6060 &

    echo "Waiting for port-foward to become available"
    sleep 5

    # Now fetch the pprof file
    curl --output "$output" http://localhost:6060/debug/heap

    echo "pprof file saved: $output"

    pkill -f oc'.*port-forward.*' || true # terminate stale port forwarding from earlier runs
    pkill -9 -f oc'.*port-forward.*' || true

    $OC delete project stackrox

    if [ "$(central_deployment_count)" != 0 ]; then
        echo "Central still available. Waiting 5s"
        sleep 5
    fi
}

##################
##     Main     ##
##################

function main() {
    now=$(date "+%Y%m%d%H%M")

    local output_folder="/tmp/pprof.$now"
    local target_v=""
    local compare_path=""
    local iterations="20"

    while [[ -n "${1:-}" ]]; do
        case "${1}" in
        "--cmp")
            compare_path="${2:-}"
            shift
            ;;
        "--output")
            output_folder="${2:-}"
            shift
            ;;
        "--target")
            target_v="${2:-}"
            shift
            ;;
        "--iterations")
            iterations="${2:-}"
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

    echo "Saving pprof files to: $output_folder"
    echo "Versions to test: $target_v $compare_path"
    echo "Running $iterations iterations"

    if [ "$(central_deployment_count)" != 0 ]; then
        echo "Centreal already deployed. Make sure to have a fresh cluster without ACS installed to run this script"
        exit
    fi

    mkdir -p "$output_folder"
    f="$output_folder/$target_v.pprof"
    run_scenario "$target_v" "$iterations" "$f"

    echo "Profling saved: $f"

    ./tests/e2e/analyze-profile.sh "$f" "$compare_path"

    echo "Profiling finished"
}

main "$@"

