#!/usr/bin/env bash

# A secure store for CI artifacts

SCRIPTS_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/lib.sh
source "$SCRIPTS_ROOT/scripts/ci/lib.sh"
# shellcheck source=../../scripts/ci/gcp.sh
source "$SCRIPTS_ROOT/scripts/ci/gcp.sh"

set -euo pipefail

store_artifacts() {
    info "Storing artifacts"

    if [[ "$#" -lt 1 ]]; then
        die "missing args. usage: store_artifacts <path> [<destination>]"
    fi

    local path="$1"
    local destination="${2:-$(basename "$path")}"

    if [ -z "${path:-}" ]; then
        echo ERROR: Missing required path parameter
        exit 1
    fi

    # Some CI do a poor job with ~ expansion
    if [[ "$path" =~ ^~ ]]; then
        path="$HOME$(cut -c2- -<<< "$path")"
    fi

    if [[ ! -e "$path" ]]; then
        echo INFO: "$path" is missing, nothing to upload
        exit 0
    fi

    if [[ -d "$path" ]] && [[ -z "$(ls -A "$path")" ]]; then
        # skip empty dirs because gsutil considers this an error
        echo INFO: "$path" is empty, nothing to upload
        exit 0
    fi

    _artifacts_preamble

    local gs_destination
    gs_destination=$(get_unique_gs_destination "${destination}")

    info "Writing to $gs_destination..."
    local exitstatus=0
    local tmp_out
    tmp_out="$(mktemp)"
    gsutil -m cp -r "$path" "$gs_destination" > "${tmp_out}" 2>&1 || exitstatus=$?
    [[ $exitstatus -eq 0 ]] || { info "gsutil cp failed:"; cat "${tmp_out}"; exit $exitstatus; }
    [[ ${TEST_OUTPUT:-0} -eq 0 ]] || cat "${tmp_out}"
}

_artifacts_preamble() {
    ensure_CI
    require_executable "gsutil"

    setup_gcp
    gsutil version -l

    set_gs_path_vars
}

get_unique_gs_destination() {
    local desired_destination="$1"
    local index=1
    local destination="$GS_JOB_URL/${desired_destination}"
    while gsutil ls "$destination" > /dev/null 2>&1; do
        (( index++ ))
        destination="$GS_JOB_URL/${desired_destination}-$index"
        if [[ $index -gt 50 ]]; then
            echo ERROR: too many attempts to find a unique destination suffix
            exit 1
        fi
    done
    echo "${destination}"
}

set_gs_path_vars() {
    GS_URL="gs://stackrox-ci-artifacts"

    if is_OPENSHIFT_CI; then
        local repo
        if [[ -n "${REPO_NAME:-}" ]]; then
            # presubmit, postsubmit and batch runs
            # (ref: https://github.com/kubernetes/test-infra/blob/master/prow/jobs.md#job-environment-variables)
            repo="${REPO_NAME}"
        elif [[ -n "${JOB_SPEC:-}" ]]; then
            # periodics
            # OpenShift CI adds 'extra_refs'
            repo="$(jq -r <<<"${JOB_SPEC}" '.extra_refs[0].repo')" || die "invalid JOB_SPEC yaml"
            if [[ "$repo" == "null" ]]; then
                die "expect: repo in JOB_SEC.extra_refs[0]"
            fi
        else
            die "Expect REPO_OWNER/NAME or JOB_SPEC"
        fi
        require_environment "BUILD_ID"
        require_environment "JOB_NAME"
        local workflow_id="${PULL_PULL_SHA:-${PULL_BASE_SHA:-nightly-$(date '+%Y%m%d')}}"
        WORKFLOW_SUBDIR="${repo}/${workflow_id}"
        JOB_SUBDIR="${BUILD_ID}-${JOB_NAME}"
        GS_JOB_URL="${GS_URL}/${WORKFLOW_SUBDIR}/${JOB_SUBDIR}"
    else
        die "Support is missing for this CI environment"
    fi
}

fixup_artifacts_content_type() {

    _artifacts_preamble

    local fixups=(
        "*.log:text/plain"
    )

    for fixup in "${fixups[@]}"; do
        IFS=':' read -ra parts <<< "$fixup"
        local file_match="${parts[0]}"
        local content_type="${parts[1]}"

        gsutil -m setmeta -h "Content-Type:$content_type" "${GS_JOB_URL}/**/$file_match" || true
    done
}

make_artifacts_help() {

    _artifacts_preamble
    
    local gs_workflow_url="$GS_URL/$WORKFLOW_SUBDIR"
    local gs_job_url="$gs_workflow_url/$JOB_SUBDIR"
    local browser_url="https://console.cloud.google.com/storage/browser/stackrox-ci-artifacts"
    local browser_job_url="$browser_url/$WORKFLOW_SUBDIR/$JOB_SUBDIR"

    local help_file
    if is_OPENSHIFT_CI; then
        require_environment "ARTIFACT_DIR"
        help_file="$ARTIFACT_DIR/howto-locate-other-artifacts-summary.html"
    else
        die "This is an unsupported environment"
    fi

    cat > "$help_file" <<- EOH
        <html>
        <head>
        <title>Additional StackRox e2e artifacts</title>
        <style>
          body { color: #e8e8e8; background-color: #424242; font-family: "Roboto", "Helvetica", "Arial", sans-serif }
          a { color: #ff8caa }
          a:visited { color: #ff8caa }
        </style>
        </head>
        <body>

        Additional StackRox e2e artifacts are stored in a GCS bucket (<code>$GS_URL</code>) by the
        <code>store_artifacts</code> bash function.<br>

        There are at least two options for access:

        <h2>Option 1: gsutil cp</h2>

        Copy all artifacts for the build/job:
        <pre>gsutil -m cp -r $gs_job_url .</pre>

        or copy all artifacts for the entire workflow:
        <pre>gsutil -m cp -r $gs_workflow_url .</pre>

        Then browse files locally.

        <h2>Option 2: Browse using the Google cloud UI</h2>

        <p>Make sure to use the URL where <code>authuser</code> corresponds to your @redhat.com account.<br>
        You can check this by clicking on the user avatar in the top right corner of Google Cloud Console page
        after following the link.</p>

        <a target="_blank" href="$browser_job_url?authuser=0">authuser=0</a><br>
        <a target="_blank" href="$browser_job_url?authuser=1">authuser=1</a><br>
        <a target="_blank" href="$browser_job_url?authuser=2">authuser=2</a><br>

        <br><br>

        </body>
        </html>
EOH

    info "Artifacts are stored in a GCS bucket ($GS_URL)"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
    if [[ "$#" -lt 1 ]]; then
        die "When invoked at the command line a method is required."
    fi
    fn="$1"
    shift
    "$fn" "$@"
fi
