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

_TABLE_NAME="acs-san-stackroxci.ci_metrics.stackrox_jobs"

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
INSERT INTO ${_TABLE_NAME}
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
UPDATE ${_TABLE_NAME}
SET $update_set
WHERE id='${METRICS_JOB_ID}'
_EO_UPDATE_

    bq query --use_legacy_sql=false "$sql"
}

slack_top_10_failures() {
    local job_name_match="${1:-qa}"
    local subject="${2:-Top 10 QA E2E Test failures for the last 7 days}"
    local is_test="${3:-false}"

    local sql
    # shellcheck disable=SC2016
    sql='
SELECT
    COUNT(*) AS `#`,
    IF(LENGTH(Classname) > 20, CONCAT(RPAD(Classname, 20), "..."), Classname) AS `Suite`,
    IF(LENGTH(Name) > 40, CONCAT(RPAD(Name, 40), "..."), Name) AS `Case`
FROM
    `acs-san-stackroxci.ci_metrics.stackrox_tests__extended_view`
WHERE
    CONTAINS_SUBSTR(ShortName, "'"${job_name_match}"'")
    AND Status = "failed"
    AND NOT IsPullRequest
    AND CONTAINS_SUBSTR(JobName, "master")
    AND NOT CONTAINS_SUBSTR(JobName, "ibmcloudz")
    AND NOT CONTAINS_SUBSTR(JobName, "powervs")
    AND DATE(Timestamp) >= DATE_SUB(DATE_TRUNC(CURRENT_DATE(), WEEK(MONDAY)), INTERVAL 1 WEEK)
GROUP BY
    Classname,
    Name
ORDER BY
    COUNT(*) DESC
LIMIT
    10
'

    local data
    echo "Running query with job match name $job_name_match"
    data="$(bq --quiet --format=pretty query --use_legacy_sql=false "$sql" 2> /dev/null)" || {
        echo >&2 -e "Cannot run query:\n${sql}\nresponse:\n${data}"
        exit 1
    }

    if [[ -z "${data}" ]]; then
        data="No failures!"
    fi
    echo "$data"

    local webhook_url
    if [[ "${is_test}" == "true" ]]; then
        webhook_url="${SLACK_CI_INTEGRATION_TESTING_WEBHOOK}"
    else
        webhook_url="${SLACK_ENG_DISCUSS_WEBHOOK}"
    fi

    local body
    # shellcheck disable=SC2016
    body='
{
    "blocks": [
        {
            "type": "header",
            "text": {
                "type": "plain_text",
                "text": "'"${subject}"'"
            }
        },
        {
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": "```\n\($data)\n```"
            }
        }
    ]
}'

    echo "Posting data to slack"
    jq -n --arg data "$data" "$body" | curl -XPOST -d @- -H 'Content-Type: application/json' "$webhook_url"
}
