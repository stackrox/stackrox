#!/usr/bin/env bash

# This script is to ensure that modifications to our Konflux pipelines follow our expectations and conventions.

set -exuo pipefail

ensure_create_snapshot_runs_last() {
    pwd
    ls -lisa
    expected_runafter="$(yq '.spec.tasks[] | select(.name != "create-acs-style-snapshot") | .name' .tekton/operator-bundle-pipeline.yaml | sort)"
    actual_runafter="$(yq '.spec.tasks[] | select(.name == "create-acs-style-snapshot") | .runAfter[]' .tekton/operator-bundle-pipeline.yaml)"

    if [ "${expected_runafter}" != "${actual_runafter}" ]; then
        echo >&2 -e """
        ERROR:
        Ensure that all previous tasks in the operator-bundle pipeline are mentioned
        in the runAfter parameter for the create-acs-style-snapshot task.
        """
        exit 1
    fi
}

ensure_create_snapshot_runs_last
