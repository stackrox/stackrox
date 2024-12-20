#!/usr/bin/env bash

# This script is to ensure that modifications to our Konflux pipelines follow our expectations and conventions.
# This script is intended to be run in CI

set -euo pipefail

ensure_create_snapshot_runs_last() {
    expected_runafter="$(yq eval '.spec.tasks[] | select(.name != "create-acs-style-snapshot") | .name' .tekton/operator-bundle-pipeline.yaml | sort)"
    actual_runafter="$(yq eval '.spec.tasks[] | select(.name == "create-acs-style-snapshot") | .runAfter[]' .tekton/operator-bundle-pipeline.yaml)"

    if ! DIFF=$(diff <(echo "${expected_runafter}") <(echo "${actual_runafter}")); then
        echo >&2 -e """
            ERROR:
            Ensure that all previous tasks in the operator-bundle pipeline are mentioned
            in the runAfter parameter for the create-acs-style-snapshot task.

            This is what is different:

            $DIFF
            """
        exit 1
    fi
}

echo "Ensure that modifications to our Konflux pipelines follow our expectations and conventions"
ensure_create_snapshot_runs_last
