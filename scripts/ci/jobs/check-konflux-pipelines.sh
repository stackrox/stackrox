#!/usr/bin/env bash

# This script is to ensure that modifications to our Konflux pipelines follow our expectations and conventions.
# This script is intended to be run in CI

set -euo pipefail

ensure_create_snapshot_runs_last() {
    expected_runafter="$(yq eval '.spec.tasks[] | select(.name != "create-acs-style-snapshot") | .name' .tekton/operator-bundle-pipeline.yaml | sort)"
    actual_runafter="$(yq eval '.spec.tasks[] | select(.name == "create-acs-style-snapshot") | .runAfter[]' .tekton/operator-bundle-pipeline.yaml)"

    echo "➤ .tekton/operator-bundle-pipeline.yaml // checking create-acs-style-snapshot: task's runAfter contents shall match the expected ones (left - expected, right - actual)."
    if ! diff --side-by-side <(echo "${expected_runafter}") <(echo "${actual_runafter}"); then
        echo >&2 -e """
✗ ERROR:

The actual runAfter contents do not match the expectations.
To resolve:

1. Open .tekton/operator-bundle-pipeline.yaml and locate the create-acs-style-snapshot task
2. Update the runAfter attribute of this task to this list of all previous tasks in the pipeline (sorted alphabetically):

${expected_runafter}
        """
        exit 1
    else
        echo "✓ No diff detected."
    fi
}

echo "Ensure consistency of our Konflux pipelines."
ensure_create_snapshot_runs_last
