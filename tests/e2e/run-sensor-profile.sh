#!/usr/bin/env bash

# Runs sensor profiling

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"
# shellcheck source=../../scripts/ci/sensor-wait.sh
source "$ROOT/scripts/ci/sensor-wait.sh"
# shellcheck source=../../tests/e2e/lib.sh
source "$ROOT/tests/e2e/lib.sh"
# shellcheck source=../../tests/scripts/setup-certs.sh
source "$ROOT/tests/scripts/setup-certs.sh"

set -euo pipefail

KUBE_BURNER_URL="https://github.com/cloud-bulldozer/kube-burner/releases/download/v1.7.8/kube-burner-V1.7.8-linux-x86_64.tar.gz"

fetch_kube_burner() {
    wget "$KUBE_BURNER_URL"
    tar xvf ./kube-burner-V1.7.8-linux-x86_64.tar.gz
}

function add_profiler_comment() {
    local text
    text="${1}"
    info "Adding a comment with profiler output to PR"

    local pr_details
    local exitstatus=0
    pr_details="$(get_pr_details)" || exitstatus="$?"
    if [[ "$exitstatus" != "0" ]]; then
        echo "DEBUG: Unable to get the PR details from GitHub: $exitstatus"
        echo "DEBUG: PR details: ${pr_details}"
        info "Will continue without commenting on the PR"
        return
    fi

    # hub-comment is tied to Circle CI env
    local url
    url=$(jq -r '.html_url' <<<"$pr_details")
    export CIRCLE_PULL_REQUEST="$url"

    local sha
    sha=$(jq -r '.head.sha' <<<"$pr_details")
    sha=${sha:0:7}
    export _SHA="$sha"

    local tag
    tag="$(make tag)"
    export _TAG="$tag"

    local tmpfile
    tmpfile="$(mktemp)"
    cat >"$tmpfile" <<-EOT
## Profile analysis results
${text}
EOT

    hub-comment -type build -template-file "$tmpfile"
}

profile_test() {
    local pprof_zip_output="$1"

    require_environment "ORCHESTRATOR_FLAVOR"
    require_environment "KUBECONFIG"
    require_environment "COMPARISON_METRICS"

    export_test_environment

    setup_gcp
    setup_deployment_env false false
    remove_existing_stackrox_resources
    setup_default_TLS_certs

    deploy_stackrox

    info "Running Sensor Profiling"

    local pprof_dir
    pprof_dir=$(dirname "${pprof_zip_output}")

    fetch_kube_burner
    kube-burner ocp cluster-density-v2 --churn=false --local-indexing=true --iterations=20 --timeout=30m

    # Once kube-burner is done, fetch the pprof
    "$ROOT/scale/profiler/pprof.sh" "${pprof_dir}" "${API_ENDPOINT}" 1
    zip -r "${pprof_zip_output}" "${pprof_dir}"

    ls -l "${pprof_dir}"

    local ftext
    ftext="$(mktemp)"
    "${ROOT}/tests/e2e/analyze-profile.sh" \
        "${pprof_dir}/heap*.prof" \
        'https://codevv.com/heap-4.0-1.prof' \
        2>/dev/null > "$ftext"

    echo "Preview analysis results: $ftext"

    # 1. Get heap profile from ${pprof_dir}
    # 2. Get heap profile to compare (maybe from a static bucket or somewhere else publicly available)
    # 3. Feed both pprofs to compare script
    # 4. Call a script to comment on the PR smilar to what `add_build_comment_to_pr` is doing

    add_profiler_comment "$(cat "$ftext")"
}

profile_test "$@"
