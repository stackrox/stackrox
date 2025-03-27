#!/usr/bin/env bash

# Create metrics relating to a CI job run. 

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"

set -euo pipefail

# Possible outcome field values for prow.
export OUTCOME_PASSED="passed"
export OUTCOME_FAILED="failed"
export OUTCOME_CANCELED="canceled"

_JOBS_TABLE_NAME="acs-san-stackroxci.ci_metrics.stackrox_jobs"

create_job_record() {
    _create_job_record "$@" || {
        # Failure to gather metrics is not a test failure
        info "WARNING: Job record creation failed"
    }
}

_create_job_record() {
    info "Creating a job record for this test run"

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: create_job_record <job name> <ci system>"
    fi

    local name="$1"

    local ci_system="$2"

    local id
    if is_OPENSHIFT_CI; then
        if [[ -z "${BUILD_ID:-}" ]]; then
            info "Skipping job record for jobs without a BUILD_ID (bin, images)"
            return
        fi
        id="${BUILD_ID}"
    elif is_GITHUB_ACTIONS; then
        id="${GITHUB_RUN_ID}"
    else
        die "Support is required for a job id for this CI environment"
    fi

    # exported to handle updates and finalization
    set_ci_shared_export "METRICS_JOB_ID" "$id"

    local repo
    repo="$(get_repo_full_name)"

    local
    branch="$(get_base_ref)"

    local pr_number=""
    if is_in_PR_context; then
        pr_number="$(get_PR_number)"
    fi

    local commit_sha
    commit_sha="$(get_commit_sha)"

    bq_create_job_record "$id" "$name" "$repo" "$branch" "$pr_number" "$commit_sha" "$ci_system"
}

bq_create_job_record() {
    setup_gcp

    bq query \
        --use_legacy_sql=false \
        --parameter="id::$1" \
        --parameter="name::$2" \
        --parameter="repo::$3" \
        --parameter="branch::$4" \
        --parameter="pr_number:INTEGER:${5:-NULL}" \
        --parameter="commit_sha::$6" \
        --parameter="ci_system::$7" \
        "INSERT INTO ${_JOBS_TABLE_NAME}
            (id, name, repo, branch, pr_number, commit_sha, started_at, ci_system)
        VALUES
            (@id, @name, @repo, @branch, @pr_number, @commit_sha, CURRENT_TIMESTAMP(), @ci_system)"
}

update_job_record() {
    _update_job_record "$@" || {
        # Failure to gather metrics is not a test failure
        info "WARNING: Job record creation failed"
    }
}

_update_job_record() {
    if [[ "$#" -lt 2 ]]; then
        die "missing arg. usage: update_job_record <field name> <value> [... more fields and values ...]"
    fi

    if is_OPENSHIFT_CI && [[ -z "${BUILD_ID:-}" ]]; then
        info "Skipping job record update for jobs without a BUILD_ID (bin, images)"
        return
    fi

    if [[ -z "${METRICS_JOB_ID:-}" ]]; then
        info "WARNING: Skipping job record update as no initial record was created"
        return
    fi

    bq_update_job_record "$@"
}

bq_update_job_record() {
    setup_gcp

    local -a sql_params
    sql_params=("--parameter=id::${METRICS_JOB_ID}")

    local update_set=""
    while [[ "$#" -ne 0 ]]; do
        local field="$1"
        local value="$2"
        shift; shift

        if [[ -n "$update_set" ]]; then
            update_set="$update_set, "
        fi

        if [[ "$field" == "stopped_at" ]]; then
            # $value is ignored, we know what to do.
            update_set="$update_set stopped_at=CURRENT_TIMESTAMP()"
        else
            update_set="$update_set $field=@$field"
            sql_params+=("--parameter=${field}::$value")
        fi
    done

    bq query \
        --use_legacy_sql=false \
        "${sql_params[@]}" \
        "UPDATE ${_JOBS_TABLE_NAME}
        SET $update_set
        WHERE id=@id"
}

slack_top_n_failures() {
    local n="${1:-10}"
    local job_name_match="${2:-qa}"
    local subject="${3:-Top 10 QA E2E Test failures for the last 7 days}"
    local is_test="${4:-false}"

    local sql
    # shellcheck disable=SC2016
    sql='
SELECT
    FORMAT("%6.2f", 100 * COUNTIF(Status="failed") / COUNT(*)) AS `%`,
    IF(LENGTH(REPLACE(Classname, "github.com/stackrox/rox/", "")) > 28,
        CONCAT(RPAD(REPLACE(Classname, "github.com/stackrox/rox/", ""), 25), "..."),
        REPLACE(Classname, "github.com/stackrox/rox/", "")) AS `Suite`,
    IF(LENGTH(Name) > 123, CONCAT(RPAD(Name, 120), "..."), Name) AS `Case`
FROM
    `acs-san-stackroxci.ci_metrics.stackrox_tests__extended_view`
WHERE
    CONTAINS_SUBSTR(ShortJobName, @job_name_match)
    -- omit PR check jobs
    AND NOT IsPullRequest
    AND NOT STARTS_WITH(JobName, "rehearse-")
    -- omit jobs on release branches
    AND NOT CONTAINS_SUBSTR(JobName, "-release-")
    -- omit jobs not owned by ACS team
    AND NOT CONTAINS_SUBSTR(JobName, "-ibmcloudz-")
    AND NOT CONTAINS_SUBSTR(JobName, "-powervs-")
    AND NOT CONTAINS_SUBSTR(JobName, "-interop-")
    -- recent
    AND DATE(Timestamp) >= DATE_SUB(DATE_TRUNC(CURRENT_DATE(), WEEK(MONDAY)), INTERVAL 1 WEEK)
GROUP BY
    Classname,
    Name
HAVING
    COUNTIF(Status="failed") > 0
ORDER BY
    COUNTIF(Status="failed") DESC
LIMIT
    @limit
'

    local data_file
    data_file="$(mktemp)"
    echo "Running query with job match name $job_name_match"
    bq --quiet --format=json query \
        --use_legacy_sql=false \
        --parameter="job_name_match::${job_name_match}" \
        --parameter="limit:INTEGER:${n}" \
        "$sql" > "${data_file}" 2>/dev/null || {
        echo >&2 -e "Cannot run query:\n${sql}\nresponse:\n$(jq < "${data_file}")"
        exit 1
    }

    local body
    if [[ $(cat "${data_file}") != "[]" ]]; then
        jq < "${data_file}"
        # shellcheck disable=SC2016
        body='{"blocks":[
            {"type": "header", "text": {"type": "plain_text", "text": "'"${subject}"'", "emoji": true}},
            {"type": "section", "fields": [
                {"type": "mrkdwn", "text": ("`Rate %` *Suite*")},
                {"type": "mrkdwn", "text": "*Case*"}
            ]},
            (.[] | {"type": "section", "fields": [
                {"type": "mrkdwn", "text": ("`"+.["%"]+"` "+.["Suite"])},
                {"type": "plain_text", "text": .["Case"]}
            ]})]}'
    else
        body='{"blocks":[
            {"type": "header", "text": {"type": "plain_text", "text": "'"${subject}"'", "emoji": true}},
            {"type": "section", "text": {"type": "plain_text", "text": "No failures! :success-kid:", "emoji": true}}]}'
        echo "No failures found!"
    fi

    echo "Posting data to slack"
    local webhook_url
    if [[ "${is_test}" == "true" ]]; then
        webhook_url="${SLACK_CI_INTEGRATION_TESTING_WEBHOOK}"
    else
        webhook_url="${SLACK_ENG_DISCUSS_WEBHOOK}"
    fi
    jq "$body" "${data_file}" | curl -XPOST -d @- -H 'Content-Type: application/json' "$webhook_url"
    rm -f "${data_file}"
}

# Saving test metrics directly after tests complete results in GCP quota issues.
# So instead they are saved to GCS and dealt with in batches as per the remedy
# in https://cloud.google.com/bigquery/docs/troubleshoot-quotas#ts-table-import-quota-resolution

_TESTS_TABLE_NAME="acs-san-stackroxci:ci_metrics.stackrox_tests"
_TESTS_STORAGE_DIR="test-metrics"

_CENTRAL_TABLE_NAME="acs-san-stackroxci:ci_metrics.stackrox_central_metrics"
_CENTRAL_STORAGE_DIR="central-metrics"

_IMAGE_PREFETCHES_TABLE_NAME="acs-san-stackroxci:ci_metrics.stackrox_image_prefetches"
_IMAGE_PREFETCHES_STORAGE_DIR="image-prefetches-metrics"

_BATCH_STORAGE_ROOT="gs://stackrox-ci-artifacts"
_BATCH_STORAGE_UPLOAD_SUBDIR="upload"

_BATCH_SIZE=20

save_test_metrics() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: save_test_metrics <CSV file>"
    fi
    _save_metrics "$1" "${_TESTS_STORAGE_DIR}"
}

save_central_metrics() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: save_central_metrics <CSV file>"
    fi
    _save_metrics "$1" "${_CENTRAL_STORAGE_DIR}"
}

save_image_prefetches_metrics() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: save_image_prefetches_metrics <CSV file>"
    fi
    _save_metrics "$1" "${_IMAGE_PREFETCHES_STORAGE_DIR}"
}

_save_metrics() {
    local csv="$1"
    local to="${_BATCH_STORAGE_ROOT}/$2/${_BATCH_STORAGE_UPLOAD_SUBDIR}"

    info "Saving Big Query test records from ${csv} to ${to}"

    gsutil cp "${csv}" "${to}/"
}

batch_load_test_metrics() {
    while _load_one_batch "${_TESTS_STORAGE_DIR}" "${_TESTS_TABLE_NAME}"; do
        info "one tests batch processed"
    done
    while _load_one_batch "${_CENTRAL_STORAGE_DIR}" "${_CENTRAL_TABLE_NAME}"; do
        info "one central batch processed"
    done
    while _load_one_batch "${_IMAGE_PREFETCHES_STORAGE_DIR}" "${_IMAGE_PREFETCHES_TABLE_NAME}"; do
        info "one image prefetches batch processed"
    done
    info "done loading"
    if [ -f error ]; then
       die "ERROR during loading one or more batch has failed"
    fi
}

_load_one_batch() {
    local files=()
    local subdir="$1"
    local table_name=$2
    info "Gathering a batch of ${subdir} to load"
    local storage_upload="${_BATCH_STORAGE_ROOT}/${subdir}/${_BATCH_STORAGE_UPLOAD_SUBDIR}"
    local storage_processing="${_BATCH_STORAGE_ROOT}/${subdir}/processing"
    local storage_done="${_BATCH_STORAGE_ROOT}/${subdir}/done"
    for metrics_file in $(gsutil ls "${storage_upload}"); do
        files+=("${metrics_file}")
        [[ "${#files[@]}" -eq "${_BATCH_SIZE}" ]] && break
    done
    if [[ "${#files[@]}" -eq 0 ]]; then
        info "No metrics found to load"
        return 1
    fi

    info "Found ${#files[@]} metric(s) for this batch load"

    # Move the batch to a new location for processing to guard against reprocess
    local process_location
    process_location="${storage_processing}/$(date +%Y-%m-%d-%H-%M-%S.%N)"
    info "Moving the batch to ${process_location}"
    gsutil -m mv "${files[@]}" "${process_location}/"
    gsutil ls -l "${process_location}"

    info "Loading into BQ"
    if bq load \
        --skip_leading_rows=1 \
        --allow_quoted_newlines \
        "$table_name" "${process_location}/*"
    then
        info "Moving the processed batch to ${storage_done}"
        gsutil -m mv "${process_location}" "${storage_done}/"
    else
        info "ERROR processing the batch, leaving in ${process_location}"
        touch error
    fi

    return 0
}
