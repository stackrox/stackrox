#!/usr/bin/env bash

# Create metrics relating to a CI job run. 

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"

set -euo pipefail

# Possible outcome field values.
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

    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: create_job_record <job name>"
    fi

    local name="$1"

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

    bq_create_job_record "$id" "$name" "$repo" "$branch" "$pr_number" "$commit_sha"
}

bq_create_job_record() {
    setup_gcp

    read -r -d '' sql <<- _EO_RECORD_ || true
INSERT INTO ${_JOBS_TABLE_NAME}
    (id, name, repo, branch, pr_number, commit_sha, started_at)
VALUES
    ('$1', '$2', '$3', '$4', ${5:-null}, '$6', CURRENT_TIMESTAMP())
_EO_RECORD_

    bq query --use_legacy_sql=false "$sql"
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

    local update_set=""
    while [[ "$#" -ne 0 ]]; do
        local field="$1"
        local value="$2"
        shift; shift

        if [[ -n "$update_set" ]]; then
            update_set="$update_set, "
        fi

        case "$field" in
            # All updateable string fields need quotation
            build|cut_*|outcome|test_target)
                value="'$value'"
                ;;
        esac

        update_set="$update_set $field=$value"
    done

    read -r -d '' sql <<- _EO_UPDATE_ || true
UPDATE ${_JOBS_TABLE_NAME}
SET $update_set
WHERE id='${METRICS_JOB_ID}'
_EO_UPDATE_

    bq query --use_legacy_sql=false "$sql"
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
    IF(LENGTH(Classname) > 28, CONCAT(RPAD(Classname, 25), "..."), Classname) AS `Suite`,
    IF(LENGTH(Name) > 123, CONCAT(RPAD(Name, 120), "..."), Name) AS `Case`
FROM
    `acs-san-stackroxci.ci_metrics.stackrox_tests__extended_view`
WHERE
    CONTAINS_SUBSTR(ShortJobName, "'"${job_name_match}"'")
    AND NOT IsPullRequest
    AND CONTAINS_SUBSTR(JobName, "master")
    AND NOT STARTS_WITH(JobName, "rehearse-")
    AND NOT CONTAINS_SUBSTR(JobName, "ibmcloudz")
    AND NOT CONTAINS_SUBSTR(JobName, "powervs")
    AND DATE(Timestamp) >= DATE_SUB(DATE_TRUNC(CURRENT_DATE(), WEEK(MONDAY)), INTERVAL 1 WEEK)
GROUP BY
    Classname,
    Name
HAVING
    COUNTIF(Status="failed") > 0
ORDER BY
    COUNTIF(Status="failed") DESC
LIMIT
    '"${n}"'
'

    local data_file
    data_file="$(mktemp)"
    echo "Running query with job match name $job_name_match"
    bq --quiet --format=json query --use_legacy_sql=false "$sql" > "${data_file}" 2>/dev/null || {
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
_BATCH_STORAGE_UPLOAD="gs://stackrox-ci-artifacts/test-metrics/upload"
_BATCH_STORAGE_PROCESSING="gs://stackrox-ci-artifacts/test-metrics/processing"
_BATCH_STORAGE_DONE="gs://stackrox-ci-artifacts/test-metrics/done"
_BATCH_SIZE=20

save_test_metrics() {
    if [[ "$#" -ne 1 ]]; then
        die "missing arg. usage: save_test_metrics <CSV file>"
    fi

    local csv="$1"
    local to="${_BATCH_STORAGE_UPLOAD}"

    info "Saving Big Query test records from ${csv} to ${to}"

    gsutil cp "${csv}" "${to}/"
}

batch_load_test_metrics() {
    while _load_one_batch; do
        info "one batch loaded"
    done
    info "done loading"
}

_load_one_batch() {
    info "Gathering a batch of test metrics to load"
    local files=()
    for metrics_file in $(gsutil ls "${_BATCH_STORAGE_UPLOAD}"); do
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
    process_location="${_BATCH_STORAGE_PROCESSING}/$(date +%Y-%m-%d-%H-%M-%S.%N)"
    info "Moving the batch to ${process_location}"
    gsutil -m mv "${files[@]}" "${process_location}/"
    gsutil ls -l "${process_location}"

    info "Loading into BQ"
    bq load \
        --skip_leading_rows=1 \
        --allow_quoted_newlines \
        "${_TESTS_TABLE_NAME}" "${process_location}/*"

    info "Moving the processed batch to ${_BATCH_STORAGE_DONE}"
    gsutil -m mv "${process_location}" "${_BATCH_STORAGE_DONE}/"

    return 0
}
