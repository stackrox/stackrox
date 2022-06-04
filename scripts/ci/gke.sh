#!/usr/bin/env bash

# A collection of GKE related reusable bash functions for CI

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"
source "$SCRIPTS_ROOT/scripts/ci/gcp.sh"

set -euo pipefail

provision_gke_cluster() {
    info "Provisioning a GKE cluster"

    setup_gcp
    assign_env_variables "$@"
    create_cluster
    if is_OPENSHIFT_CI; then
        help_for_cluster_access
    fi
}

assign_env_variables() {
    info "Assigning environment variables for later steps"

    if [[ "$#" -lt 1 ]]; then
        die "missing args. usage: assign_env_variables <cluster-id> [<num-nodes> <machine-type>]"
    fi

    local cluster_id="$1"
    local num_nodes="${2:-3}"
    local machine_type="${3:-e2-standard-4}"

    ensure_CI

    local build_num
    if is_OPENSHIFT_CI; then
        require_environment "BUILD_ID"
        build_num="${BUILD_ID}"
    elif is_CIRCLECI; then
        require_environment "CIRCLE_BUILD_NUM"
        build_num="${CIRCLE_BUILD_NUM}"
    else
        die "Support is missing for this CI environment"
    fi

    local cluster_name="rox-ci-${cluster_id}-${build_num}"
    ci_export CLUSTER_NAME "$cluster_name"
    echo "Assigned cluster name is $cluster_name"

    ci_export NUM_NODES "$num_nodes"
    echo "Number of nodes for cluster is $num_nodes"

    ci_export MACHINE_TYPE "$machine_type"
    echo "Machine type is set as to $machine_type"

    local gke_release_channel="stable"
    if is_CIRCLECI; then
        if "$SCRIPTS_ROOT/.circleci/pr_has_label.sh" ci-gke-release-channel-rapid; then
            gke_release_channel="rapid"
        elif "$SCRIPTS_ROOT/.circleci/pr_has_label.sh" ci-gke-release-channel-regular; then
            gke_release_channel="regular"
        fi
    fi
    ci_export GKE_RELEASE_CHANNEL "$gke_release_channel"
    echo "Using gke release channel: $gke_release_channel"
}

create_cluster() {
    info "Creating a GKE cluster"

    ensure_CI

    require_environment "CLUSTER_NAME"

    local tags="stackrox-ci"
    local labels="stackrox-ci=true"
    if is_OPENSHIFT_CI; then
        require_environment "JOB_NAME"
        require_environment "BUILD_ID"
        tags="${tags},stackrox-ci-${JOB_NAME:0:50}"
        tags="${tags/%-/x}"
        labels="${labels},stackrox-ci-job=${JOB_NAME:0:63}"
        labels="${labels/%-/x}"
        labels="${labels},stackrox-ci-build-id=${BUILD_ID:0:63}"
        labels="${labels/%-/x}"
    elif is_CIRCLECI; then
        require_environment "CIRCLE_JOB"
        require_environment "CIRCLE_WORKFLOW_ID"
        tags="${tags},stackrox-ci-${CIRCLE_JOB:0:50}"
        tags="${tags/%-/x}"
        labels="${labels},stackrox-ci-job=${CIRCLE_JOB:0:63}"
        labels="${labels/%-/x}"
        labels="${labels},stackrox-ci-workflow=${CIRCLE_WORKFLOW_ID:0:63}"
        labels="${labels/%-/x}"
    else
        die "Support is missing for this CI environment"
    fi
    # lowercase
    tags="${tags,,}"
    labels="${labels,,}"

    ### Network Sizing ###
    # The overall subnetwork ("--create-subnetwork") is used for nodes.
    # The "cluster" secondary range is for pods ("--cluster-ipv4-cidr").
    # The "services" secondary range is for ClusterIP services ("--services-ipv4-cidr").
    # See https://cloud.google.com/kubernetes-engine/docs/how-to/alias-ips#cluster_sizing.

    REGION=us-central1
    NUM_NODES="${NUM_NODES:-3}"
    GCP_IMAGE_TYPE="${GCP_IMAGE_TYPE:-UBUNTU}"
    POD_SECURITY_POLICIES="${POD_SECURITY_POLICIES:-false}"
    GKE_RELEASE_CHANNEL="${GKE_RELEASE_CHANNEL:-stable}"
    MACHINE_TYPE="${MACHINE_TYPE:-e2-standard-4}"

    echo "Creating ${NUM_NODES} node cluster with image type \"${GCP_IMAGE_TYPE}\""

    VERSION_ARGS=(--release-channel "${GKE_RELEASE_CHANNEL}")
    get_supported_cluster_version
    if [[ -n "${CLUSTER_VERSION:-}" ]]; then
        echo "using cluster version: ${CLUSTER_VERSION}"
        VERSION_ARGS=(--cluster-version "${CLUSTER_VERSION}")
    fi

    PSP_ARG=
    if [[ "${POD_SECURITY_POLICIES}" == "true" ]]; then
        PSP_ARG="--enable-pod-security-policy"
    fi
    zones=$(gcloud compute zones list --filter="region=$REGION" | grep UP | cut -f1 -d' ' | shuf)
    success=0
    for zone in $zones; do
        if is_CIRCLECI; then
            "$SCRIPTS_ROOT/.circleci/check-workflow-live.sh" || return 1
        fi
        echo "Trying zone $zone"
        ci_export ZONE "$zone"
        gcloud config set compute/zone "${zone}"
        status=0
        # shellcheck disable=SC2153
        timeout 630 gcloud beta container clusters create \
            --machine-type "${MACHINE_TYPE}" \
            --num-nodes "${NUM_NODES}" \
            --disk-type=pd-standard \
            --disk-size=40GB \
            --create-subnetwork range=/28 \
            --cluster-ipv4-cidr=/20 \
            --services-ipv4-cidr=/24 \
            --enable-ip-alias \
            --enable-network-policy \
            --enable-autorepair \
            "${VERSION_ARGS[@]}" \
            --image-type "${GCP_IMAGE_TYPE}" \
            --tags="${tags}" \
            --labels="${labels}" \
            ${PSP_ARG} \
            "${CLUSTER_NAME}" || status="$?"
        if [[ "${status}" == 0 ]]; then
            success=1
            break
        elif [[ "${status}" == 124 ]]; then
            echo >&2 "gcloud command timed out. Checking to see if cluster is still creating"
            if ! gcloud container clusters describe "${CLUSTER_NAME}" >/dev/null; then
                echo >&2 "Create cluster did not create the cluster in Google. Trying a different zone..."
            else
                for i in {1..60}; do
                    if [[ "$(gcloud container clusters describe "${CLUSTER_NAME}" --format json | jq -r .status)" == "RUNNING" ]]; then
                        success=1
                        break
                    fi
                    sleep 20
                    echo "Currently have waited $((i * 5)) for cluster ${CLUSTER_NAME} in ${zone} to move to running state"
                done
            fi

            if [[ "${success}" == 1 ]]; then
                echo "Successfully launched cluster ${CLUSTER_NAME}"
                break
            fi
            echo >&2 "Timed out after 10 more minutes. Trying another zone..."
            echo >&2 "Deleting the cluster"
            gcloud container clusters delete "${CLUSTER_NAME}" --async
        fi
    done

    if [[ "${success}" == "0" ]]; then
        echo "Cluster creation failed"
        return 1
    fi
}

wait_for_cluster() {
    info "Waiting for a GKE cluster to stabilize"

    while [[ $(kubectl -n kube-system get pod | tail -n +2 | wc -l) -lt 2 ]]; do
        echo "Still waiting for kubernetes to create initial kube-system pods"
        sleep 1
    done

    local grace_period=30
    while true; do
        kubectl -n kube-system get pod
        local numstarting
        numstarting=$(kubectl -n kube-system get pod -o json | jq '[(.items[].status.containerStatuses // [])[].ready | select(. | not)] | length')
        if ((numstarting == 0)); then
            local last_start_ts
            last_start_ts="$(kubectl -n kube-system get pod -o json | jq '[(.items[].status.containerStatuses // [])[] | (.state.running.startedAt // (now | todate)) | fromdate] | max')"
            local curr_ts
            curr_ts="$(date '+%s')"
            local remaining_grace_period
            remaining_grace_period=$((last_start_ts + grace_period - curr_ts))
            if ((remaining_grace_period <= 0)); then
                break
            fi
            echo "Waiting for another $remaining_grace_period seconds for kube-system pods to stabilize"
            sleep "$remaining_grace_period"
        fi

        echo "Waiting for ${numstarting} kube-system containers to be initialized"
        sleep 10
    done
}

get_supported_cluster_version() {
    if [[ -n "${CLUSTER_VERSION:-}" ]]; then
        local match
        match=$(gcloud container get-server-config --format json | jq "[.validMasterVersions | .[] | select(.|test(\"^${CLUSTER_VERSION}\"))][0]")
        if [[ -z "${match}" || "${match}" == "null" ]]; then
            echo "A supported version cannot be found that matches ${CLUSTER_VERSION}."
            echo "Valid master versions are:"
            gcloud container get-server-config --format json | jq .validMasterVersions
            exit 1
        fi
        CLUSTER_VERSION=$(sed -e 's/^"//' -e 's/"$//' <<<"${match}")
    fi
}

refresh_gke_token() {
    info "Starting a GKE token refresh loop"

    require_environment "ZONE"
    require_environment "CLUSTER_NAME"

    local real_kubeconfig="${KUBECONFIG:-${HOME}/.kube/config}"

    # refresh token every 15m
    while true; do
        # sleep & wait so that it will exit on TERM
        sleep 900 &
        wait $!
        info "Refreshing the GKE auth token"
        gcloud config config-helper --force-auth-refresh >/dev/null
        echo >/tmp/kubeconfig-new
        chmod 0600 /tmp/kubeconfig-new
        # shellcheck disable=SC2153
        KUBECONFIG=/tmp/kubeconfig-new gcloud container clusters get-credentials --project stackrox-ci --zone "$ZONE" "$CLUSTER_NAME"
        KUBECONFIG=/tmp/kubeconfig-new kubectl get ns >/dev/null
        mv /tmp/kubeconfig-new "$real_kubeconfig"
    done
}

teardown_gke_cluster() {
    info "Tearing down the GKE cluster: ${CLUSTER_NAME:-}"

    require_environment "CLUSTER_NAME"
    require_executable "gcloud"

    # (prefix output to avoid triggering prow log focus)
    "$SCRIPTS_ROOT/scripts/ci/cleanup-deployment.sh" 2>&1 | sed -e 's/^/out: /' || true

    gcloud container clusters delete "$CLUSTER_NAME" --async

    info "Cluster deleting asynchronously"
}

help_for_cluster_access() {
    local help_file="$ARTIFACT_DIR/cluster-access-summary.html"
    local project
    project="$(gcloud config get-value project)"

    cat > "$help_file" <<- EOH
<html>
    <head>
        <title><h4>E2e Test Cluster Access</h4></title>
    </head>
    <body>
        <style>
        /* style for prow spyglass html lens */
        #wrapper {
            color: #fff !important;
        }
        </style>

        <p>If you have the required GCP account privilege you can connect to the cluster in use in this test with:</p>

        <pre>gcloud container clusters get-credentials "${CLUSTER_NAME}" --project "${project}" --zone "${ZONE}"</pre>

        <br>
        <br>
    </body>
</html>
EOH
}


if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
