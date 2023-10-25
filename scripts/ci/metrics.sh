#!/usr/bin/env bash

# Create metrics relating to a CI job run. 

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
# shellcheck source=../../scripts/ci/gcp.sh
source "$ROOT/scripts/ci/gcp.sh"

set -euo pipefail

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
    export METRICS_JOB_ID="$id"

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

finalize_job_record() {
    _finalize_job_record "$@" || {
        # Failure to gather metrics is not a test failure
        info "WARNING: Job record creation failed"
    }
}

_finalize_job_record() {
    info "Finalizing a job record for this test run"

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: finalize_job_record <exit code> <canceled (true|false)"
    fi

    if is_OPENSHIFT_CI && [[ -z "${BUILD_ID:-}" ]]; then
        info "Skipping job record finalization for jobs without a BUILD_ID (bin, images)"
        return
    fi

    if [[ -z "${METRICS_JOB_ID:-}" ]]; then
        info "WARNING: Skipping job record finalization as no initial record was created"
        return
    fi

    local sql
    read -r -d '' sql <<- _EO_CHECK_ || true
SELECT outcome 
FROM ${_TABLE_NAME}
WHERE id='${METRICS_JOB_ID}'
_EO_CHECK_

    local outcome
    outcome="$(bq --quiet --format=json query --use_legacy_sql=false "$sql" | jq -r '.[0].outcome')"

    if [[ "$outcome" != "null" ]]; then
        info "WARNING: This jobs record is already finalized ($outcome), will not overwrite"
        return
    fi

    local exit_code="$1"
    local canceled="$2"

    if [[ "$canceled" == "true" ]]; then
        outcome="canceled"
    elif [[ "$exit_code" == "0" ]]; then
        outcome="passed"
    else
        outcome="failed"
    fi

    update_job_record outcome "$outcome" stopped_at "CURRENT_TIMESTAMP()"
}
