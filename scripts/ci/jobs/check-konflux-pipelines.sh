#!/usr/bin/env bash

# This script is to ensure that modifications to our Konflux pipelines follow our expectations and conventions.
# This script is intended to be run in CI

set -euo pipefail

ensure_create_snapshot_runs_last() {
    local pipeline_path=".tekton/operator-bundle-pipeline.yaml"
    local task_name="create-acs-style-snapshot"
    expected_runafter="$(yq eval '.spec.tasks[] | select(.name != '\"${task_name}\"') | .name' "${pipeline_path}" | sort)"
    actual_runafter="$(yq eval '.spec.tasks[] | select(.name == '\"${task_name}\"') | .runAfter[]' "${pipeline_path}")"

    echo "➤ ${pipeline_path} // checking ${task_name}: task's runAfter contents shall match the expected ones (left - expected, right - actual)."
    if ! diff --side-by-side <(echo "${expected_runafter}") <(echo "${actual_runafter}"); then
        echo >&2 -e "
✗ ERROR:

The actual runAfter contents do not match the expectations.
To resolve:

1. Open ${pipeline_path} and locate the ${task_name} task
2. Update the runAfter attribute of this task to this list of all previous tasks in the pipeline (sorted alphabetically):

${expected_runafter}
        "
        exit 1
    else
        echo "✓ No diff detected."
    fi
}

echo "Ensure consistency of our Konflux pipelines."
ensure_create_snapshot_runs_last
