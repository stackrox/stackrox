#!/usr/bin/env bash

# shellcheck disable=SC2329 # dispatch.sh calls this function after sourcing the file.
ci_decision_allows() {
    local status=0
    get_pr_details | CI_SELECT_ROOT="$ROOT" "$ROOT/ci/select/bin/ci-select" prow-ci "${1:?job name}" || status=$?
    # Only an explicit skip (3) stops the job; setup and API failures let it run.
    [[ "$status" -ne 3 ]]
}
