#!/usr/bin/env bash

# This script is to ensure that modifications to our Konflux pipelines follow our expectations and conventions.
# This script is intended to be run in CI

set -euo pipefail

FAIL_FLAG="$(mktemp)"
trap 'rm -f $FAIL_FLAG' EXIT

check_create_snapshot_runs_last() {
    local -r pipeline_path=".tekton/operator-bundle-pipeline.yaml"
    local -r task_name="create-acs-style-snapshot"

    local expected_runafter
    expected_runafter="$(yq eval '.spec.tasks[] | select(.name != '\"${task_name}\"') | .name' "${pipeline_path}" | sort)"

    local actual_runafter
    actual_runafter="$(yq eval '.spec.tasks[] | select(.name == '\"${task_name}\"') | .runAfter[]' "${pipeline_path}")"

    echo
    echo "➤ ${pipeline_path} // checking ${task_name}: task's runAfter contents shall match the expected ones."
    if ! compare "${expected_runafter}" "${actual_runafter}"; then
        echo >&2 "::error file=${pipeline_path}::${task_name} runAfter is not expected
it should be:
${expected_runafter}" # multiline
        echo >&2 -e "How to resolve:
1. Open ${pipeline_path} and locate the ${task_name} task
2. Update the runAfter attribute of this task to this list of all previous tasks in the pipeline (sorted alphabetically):
${expected_runafter}"
        record_failure "${FUNCNAME}"
    fi
}

check_all_components_are_part_of_custom_snapshot() {
    local -r pipeline_path=".tekton/operator-bundle-pipeline.yaml"
    local -r task_name="create-acs-style-snapshot"

    # Actual components are based on the COMPONENTS parameter and stored as sorted multi-line string.
    local actual_components
    actual_components="$(yq eval '.spec.tasks[] | select(.name == '\"${task_name}\"') | .params[] | select(.name == "COMPONENTS") | .value' "${pipeline_path}" | yq eval '.[].name' - | tr " " "\n" | sort)"

    # Expected components are based on the wait-for-*-image task plus the operator-bundle and stored as a sorted multi-line string.
    local expected_components
    expected_components_from_images="$(yq eval '.spec.tasks[] | select(.name == "wait-for-*-image") | .name | sub("(wait-for-|-image)", "")' ${pipeline_path})"
    expected_components=$(echo "${expected_components_from_images} operator-bundle" | tr " " "\n" | sort)

    echo
    echo "➤ ${pipeline_path} // checking ${task_name}: COMPONENTS contents shall include all ACS images."
    if ! compare "${expected_components}" "${actual_components}"; then
        echo >&2 "::error file=${pipeline_path}::${task_name} COMPONENTS are not expected" # to stderr
        echo >&2 -e "How to resolve:
1. Open ${pipeline_path} and locate the ${task_name} task
2. Update the COMPONENTS parameter of this task to include entries for the missing components or delete references to removed components. COMPONENTS should include entries for (sorted alphabetically):
${expected_components}"
        record_failure "${FUNCNAME}"
    fi
}

check_test_rpmdb_files_are_ignored() {
    # At the time of this writing, Konflux uses syft to generate SBOMs for built containers.
    # If we happen to have test rpmdb databases in the repo, syft will union their contents with RPMs that it finds
    # installed in the container resulting in a misleading SBOM.
    # This check is to make sure we list all such rpmdbs in the ignore list in Syft's config.
    # Ref https://github.com/anchore/syft/wiki/configuration

    local -r syft_config=".syft.yaml"
    local -r exclude_attribute=".exclude"

    local actual_excludes
    actual_excludes="$(yq eval "${exclude_attribute}" "${syft_config}")"

    local expected_excludes
    expected_excludes="$(git ls-files -- '**/rpmdb.sqlite' | sort | uniq | sed 's/^/- .\//')"

    echo
    echo "➤ ${syft_config} // checking ${exclude_attribute}: all rpmdb files in the repo shall be mentioned."
    if ! compare "${expected_excludes}" "${actual_excludes}"; then
        echo "::error file=${syft_config}::${exclude_attribute} contents are not expected"
        echo >&2 "How to resolve:
1. Open ${syft_config} and replace ${exclude_attribute} contents with the following.
${expected_excludes}"
        record_failure "${FUNCNAME}"
    fi
}

compare() {
    local -r expected="$1"
    local -r actual="$2"

    if ! diff --brief <(echo "${expected}") <(echo "${actual}") > /dev/null; then
        echo >&2 "✗ ERROR: the expected contents (left) don't match the actual ones (right):"
        diff >&2 --side-by-side <(echo "${expected}") <(echo "${actual}") || true
        return 1
    else
        echo "✓ No diff detected."
    fi
}

record_failure() {
    local -r func="$1"
    echo "${func}" >> "${FAIL_FLAG}"
}

echo "Checking our Konflux pipelines and builds setup."
check_create_snapshot_runs_last
check_all_components_are_part_of_custom_snapshot
check_test_rpmdb_files_are_ignored

if [[ -s "$FAIL_FLAG" ]]; then
    echo >&2
    echo >&2 "✗ Some Konflux pipeline consistency checks failed:"
    cat >&2 "$FAIL_FLAG"
    exit 1
else
    echo
    echo "✓ All checks passed."
fi
