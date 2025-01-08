#!/usr/bin/env bash

# This script is to ensure that modifications to our Konflux pipelines follow our expectations and conventions.
# This script is intended to be run in CI

set -euo pipefail

FAIL_FLAG="/tmp/fail"

ensure_create_snapshot_runs_last() {
    local pipeline_path=".tekton/operator-bundle-pipeline.yaml"
    local task_name="create-acs-style-snapshot"
    expected_runafter="$(yq '.spec.tasks[] | select(.name != '\"${task_name}\"') | .name' "${pipeline_path}" | sort)"
    actual_runafter="$(yq '.spec.tasks[] | select(.name == '\"${task_name}\"') | .runAfter[]' "${pipeline_path}")"

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
    # I realized one other thing.
    # We check that create-acs-style-snapshot runs last but we don't have a check that all product's Components are passed to the Snapshot.
    # Imagine when we introduce a new container. We'll add it to Helm charts, we'll add it to the Operator bundle but we may forget to add
    # it to the Snapshot creation because not many folks besides us three know what Snapshots are and what role they play.

    # I'd suggest to extend check-konflux-pipelines.sh to validate that.
    # Maybe it's best to do in a follow-up PR/JIRA task to not delay this PR #13577 for even longer.
    # Besides unwrapping a string from COMPONENTS value (YAML is a superset of JSON so I think yq can parse COMPONENTS raw value),
    # the tricky thing is what to compare that to.
    # I suggest that we can compare components names with wait-for-*-image tasks in the pipeline.
    # The idea is that folks will likely remember to add the new image to the operator-bundle by the time of release and the
    # check will remind them to include it in the Snapshot too.

    actual_components="$(yq '.spec.tasks[] | select(.name == '\"${task_name}\"') | .params[] | select(.name == "COMPONENTS") | .value' "${pipeline_path}" | yq '.[].name' | tr " " "\n" | sort)"
    expected_components_from_images="$(yq '.spec.tasks[] | select(.name == "wait-for-*-image") | .name | sub("(wait-for-|-image)", "")' .tekton/operator-bundle-pipeline.yaml)"
    expected_components=$(echo "${expected_components_from_images} operator-bundle" | tr " " "\n" | sort)

    echo "➤ ${pipeline_path} // checking ${task_name}: COMPONENTS contents shall include all ACS images (left - expected, right - actual)."
    if ! diff --side-by-side <(echo "${expected_components}") <(echo "${actual_components}"); then
        echo >&2 -e "
✗ ERROR:

The actual COMPONENTS contents do not match the expectations.
To resolve:

1. Open ${pipeline_path} and locate the ${task_name} task
2. Update the COMPONENTS parameter of this task to include entries for the missing components or remove references to deprecated components. COMPONENTS should include entries for (sorted alphabetically):

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

if [[ -e "$FAIL_FLAG" ]]; then
    echo "ERROR: Some Konflux pipeline consistency checks failed:"
    cat "$FAIL_FLAG"
    exit 1
fi
