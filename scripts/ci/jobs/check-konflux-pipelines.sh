#!/usr/bin/env bash

# This script is to ensure that modifications to our Konflux pipelines follow our expectations and conventions.
# This script is intended to be run in CI

set -euo pipefail

FAIL_FLAG="$(mktemp)"
trap 'rm -f $FAIL_FLAG' EXIT

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
        return 1
    else
        echo "✓ No diff detected."
    fi
}

check_all_components_part_of_custom_snapshot() {
    local pipeline_path=".tekton/operator-bundle-pipeline.yaml"
    local task_name="create-acs-style-snapshot"

    # Actual components are based on the COMPONENTS parameter and stored as sorted multi-line string.
    actual_components="$(yq eval '.spec.tasks[] | select(.name == '\"${task_name}\"') | .params[] | select(.name == "COMPONENTS") | .value' "${pipeline_path}" | yq eval '.[].name' - | tr " " "\n" | sort)"
    # Expected components are based on the wait-for-*-image task plus the operator-bundle and stored as a sorted multi-line string.
    expected_components_from_images="$(yq eval '.spec.tasks[] | select(.name == "wait-for-*-image") | .name | sub("(wait-for-|-image)", "")' ${pipeline_path})"
    expected_components=$(echo "${expected_components_from_images} operator-bundle" | tr " " "\n" | sort)

    echo "➤ ${pipeline_path} // checking ${task_name}: COMPONENTS contents shall include all ACS images (left - expected, right - actual)."
    if ! diff --side-by-side <(echo "${expected_components}") <(echo "${actual_components}"); then
        echo >&2 -e "
✗ ERROR:

The actual COMPONENTS contents do not match the expectations.
To resolve:

1. Open ${pipeline_path} and locate the ${task_name} task
2. Update the COMPONENTS parameter of this task to include entries for the missing components or delete references to removed components. COMPONENTS should include entries for (sorted alphabetically):

${expected_components}
        "
        return 1
    else
        echo "✓ No diff detected."
    fi
}

echo "Ensure consistency of our Konflux pipelines."
ensure_create_snapshot_runs_last || { echo "ensure_create_snapshot_runs_last" >> "${FAIL_FLAG}"; }
check_all_components_part_of_custom_snapshot || { echo "check_all_components_part_of_custom_snapshot" >> "${FAIL_FLAG}"; }

if [[ -s "$FAIL_FLAG" ]]; then
    echo >&2 "ERROR: Some Konflux pipeline consistency checks failed:"
    cat >&2 "$FAIL_FLAG"
    exit 1
fi
