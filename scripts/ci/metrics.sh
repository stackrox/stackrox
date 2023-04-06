#!/usr/bin/env bash

# Create metrics relating to a CI job run. 

set -euo pipefail

_TABLE_NAME="stackrox-ci.ci_metrics.stackrox_jobs"

create_job_record() {
    _create_job_record "$@" || {
        # ROX-xyz ignore initial job record failures until all environments are
        # stable (PR, merge, nightly, release, tag, etc)
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
        id="${BUILD_ID}"
    elif is_GITHUB_ACTIONS; then
        id="${GITHUB_RUN_ID}"
    else
        die "Support is required for a job id for this CI environment"
    fi

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
        # ROX-xyz ignore initial job record failures until all environments are
        # stable (PR, merge, nightly, release, tag, etc)
        info "WARNING: Job record creation failed"
    }
}

_update_job_record() {
    if [[ "$#" -lt 2 ]]; then
        die "missing arg. usage: update_job_record <field name> <value> [... more fields and values ...]"
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
            outcome)
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
        # ROX-xyz ignore initial job record failures until all environments are
        # stable (PR, merge, nightly, release, tag, etc)
        info "WARNING: Job record creation failed"
    }
}

_finalize_job_record() {
    info "Finalizing a job record for this test run"

    if [[ "$#" -ne 2 ]]; then
        die "missing arg. usage: finalize_job_record <exit code> <canceled (true|false)"
    fi

    local outcome
    outcome="$(bq --quiet --format=json query --use_legacy_sql=false \
        'select outcome from stackrox-ci.ci_metrics.stackrox_jobs' \
        | jq -r '.[0].outcome')"

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
