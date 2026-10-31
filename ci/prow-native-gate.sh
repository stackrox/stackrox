#!/usr/bin/env bash
# GENERATED FILE - DO NOT EDIT
# Generated from: ci/test-selection-rules.yaml
# To regenerate: make generate-ci-selection
#
# Usage: Source this file in dispatch.sh and call ci_should_run_job <job-name>
# Returns 0 if job should run, 1 if job should skip

set -euo pipefail

# Get changed files between base and head
get_changed_files() {
    if [[ -z "${PULL_BASE_SHA:-}" ]] || [[ -z "${PULL_PULL_SHA:-}" ]]; then
        return 1
    fi
    git diff --name-only "${PULL_BASE_SHA}...${PULL_PULL_SHA}" 2>/dev/null || return 1
}

# Get PR labels via GitHub API
get_pr_labels() {
    if [[ -z "${PULL_NUMBER:-}" ]]; then
        return 1
    fi
    local repo="${REPO_OWNER:-stackrox}/${REPO_NAME:-stackrox}"
    gh pr view "${PULL_NUMBER}" --repo "${repo}" --json labels --jq '.labels[].name' 2>/dev/null || return 1
}

# Check if /test override is present for this job
has_test_override() {
    local job="$1"

    if [[ -z "${PULL_NUMBER:-}" ]] || [[ -z "${PULL_PULL_SHA:-}" ]]; then
        return 1
    fi

    local repo="${REPO_OWNER:-stackrox}/${REPO_NAME:-stackrox}"
    local commit_time
    commit_time="$(git show -s --format=%cI "${PULL_PULL_SHA}" 2>/dev/null)" || return 1

    # Get comments after commit time
    local comments
    comments="$(gh api --paginate "repos/${repo}/issues/${PULL_NUMBER}/comments" \
        --jq ".[] | select(.created_at >= \"${commit_time}\") | .body" 2>/dev/null)" || return 1

    # Check for /test <job> or /test all
    if grep -qE "^/test (${job}|all)$" <<< "${comments}"; then
        return 0
    fi

    return 1
}

# Check if label exists in labels list
has_label() {
    local label="$1"
    local labels="$2"
    grep -qxF "${label}" <<< "${labels}"
}

# Check if any file matches any of the patterns (extended regex)
any_file_matches() {
    local files="$1"
    shift
    local patterns=("$@")

    for pattern in "${patterns[@]}"; do
        if grep -qE "${pattern}" <<< "${files}"; then
            return 0
        fi
    done
    return 1
}

# Check if every file matches at least one pattern (extended regex)
every_file_matches() {
    local files="$1"
    shift
    local patterns=("$@")

    [[ -z "${files}" ]] && return 1

    while IFS= read -r file; do
        local matched=false
        for pattern in "${patterns[@]}"; do
            if grep -qE "${pattern}" <<< "${file}"; then
                matched=true
                break
            fi
        done
        if [[ "${matched}" == "false" ]]; then
            return 1
        fi
    done <<< "${files}"

    return 0
}

# Main decision function
ci_should_run_job() {
    local job="$1"

    # Fail-open: if we can't determine, run the job
    local changed_files
    if ! changed_files="$(get_changed_files)"; then
        echo "Warning: Cannot determine changed files, running job ${job}" >&2
        return 0
    fi

    local pr_labels
    if ! pr_labels="$(get_pr_labels)"; then
        echo "Warning: Cannot read PR labels, running job ${job}" >&2
        return 0
    fi

    # Check for /test override first (highest priority)
    if has_test_override "${job}"; then
        echo "Info: /test override detected for ${job}" >&2
        return 0
    fi

    # Track votes from rules
    local vote_run=false
    local vote_skip=false

    # Apply rules in order

    # Rule: ci-run-all-tests-label - Force run all tests when label present
    if has_label "ci-run-all-tests" "${pr_labels}"; then
        if [[ "${job}" == "style-check" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "check-generated-files" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go-postgres" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "shell-unit-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui-component" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "golangci-lint" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "check-crd-compatibility" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "db-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-qa-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-upgrade-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-nongroovy-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ocp-qa-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests-openshift" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "wait-for-images" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "should-dispatch" ]]; then
            vote_run=true
        fi
    fi

    # Rule: area-sensor-label - Run sensor-related tests when area/sensor label present
    if has_label "area/sensor" "${pr_labels}"; then
        if [[ "${job}" == "sensor-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests-openshift" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
    fi

    # Rule: area-central-label - Run central-related tests when area/central label present
    if has_label "area/central" "${pr_labels}"; then
        if [[ "${job}" == "go-postgres" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
    fi

    # Rule: area-ui-label - Run UI tests when area/ui label present
    if has_label "area/ui" "${pr_labels}"; then
        if [[ "${job}" == "ui" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui-component" ]]; then
            vote_run=true
        fi
    fi

    # Rule: go-mod-changes - go.mod/go.sum changes affect all Go code
    if any_file_matches "${changed_files}" "^go\.mod$" "^go\.sum$"; then
        if [[ "${job}" == "style-check" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "check-generated-files" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go-postgres" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "shell-unit-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui-component" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "golangci-lint" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "check-crd-compatibility" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "db-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-qa-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-upgrade-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-nongroovy-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ocp-qa-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests-openshift" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "wait-for-images" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "should-dispatch" ]]; then
            vote_run=true
        fi
    fi

    # Rule: proto-changes - Proto changes affect generated code across codebase
    if any_file_matches "${changed_files}" "^proto/"; then
        if [[ "${job}" == "style-check" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "check-generated-files" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go-postgres" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "shell-unit-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui-component" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "golangci-lint" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "check-crd-compatibility" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "db-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-qa-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-upgrade-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-nongroovy-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ocp-qa-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests-openshift" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "wait-for-images" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "should-dispatch" ]]; then
            vote_run=true
        fi
    fi

    # Rule: generated-code-changes - Generated code changes need full validation
    if any_file_matches "${changed_files}" "^generated/"; then
        if [[ "${job}" == "style-check" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "check-generated-files" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go-postgres" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "shell-unit-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui-component" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "golangci-lint" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "check-crd-compatibility" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "db-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-qa-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-upgrade-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-nongroovy-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ocp-qa-e2e-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests-openshift" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "wait-for-images" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "should-dispatch" ]]; then
            vote_run=true
        fi
    fi

    # Rule: docs-only-skip - Skip most tests for docs-only changes
    if every_file_matches "${changed_files}" "\.md$" "^docs/" "^CHANGELOG"; then
        if [[ "${job}" == "check-generated-files" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "go-postgres" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "sensor-integration-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "shell-unit-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "ui" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "ui-component" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "golangci-lint" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "check-crd-compatibility" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "db-integration-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "gke-qa-e2e-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "gke-upgrade-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "gke-nongroovy-e2e-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "ocp-qa-e2e-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "sensor-integration-tests-openshift" ]]; then
            vote_skip=true
        fi
    fi

    # Rule: ui-only-skip-backend - Skip backend tests for UI-only changes
    if every_file_matches "${changed_files}" "^ui/"; then
        if [[ "${job}" == "ui" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui-component" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "style-check" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go-postgres" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "sensor-integration-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "sensor-integration-tests-openshift" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "gke-qa-e2e-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "gke-upgrade-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "ocp-qa-e2e-tests" ]]; then
            vote_skip=true
        fi
    fi

    # Rule: sensor-subsystem - Sensor code changes need sensor tests
    if any_file_matches "${changed_files}" "^sensor/" "^pkg/sensor/"; then
        if [[ "${job}" == "sensor-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go-postgres" ]]; then
            vote_skip=true
        fi
    fi

    # Rule: central-subsystem - Central code changes need central tests
    if any_file_matches "${changed_files}" "^central/" "^pkg/central/"; then
        if [[ "${job}" == "go-postgres" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "sensor-integration-tests" ]]; then
            vote_skip=true
        fi
    fi

    # Rule: scanner-subsystem - Scanner code changes
    if any_file_matches "${changed_files}" "^scanner/" "^pkg/scanner/"; then
        if [[ "${job}" == "db-integration-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
    fi

    # Rule: operator-subsystem - Operator code changes
    if any_file_matches "${changed_files}" "^operator/"; then
        if [[ "${job}" == "check-crd-compatibility" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
    fi

    # Rule: ci-tooling-changes - CI tooling changes need validation but can skip E2E
    if any_file_matches "${changed_files}" "^\.github/workflows/" "^\.openshift-ci/" "^ci/select/" "^scripts/ci/"; then
        if [[ "${job}" == "style-check" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "shell-unit-tests" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "gke-qa-e2e-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "gke-upgrade-tests" ]]; then
            vote_skip=true
        fi
        if [[ "${job}" == "ocp-qa-e2e-tests" ]]; then
            vote_skip=true
        fi
    fi

    # Rule: go-sources - Any .go file change needs Go tests
    if any_file_matches "${changed_files}" "\.go$"; then
        if [[ "${job}" == "go" ]]; then
            vote_run=true
        fi
    fi

    # Rule: shell-scripts - Shell script changes need shell tests
    if any_file_matches "${changed_files}" "\.sh$"; then
        if [[ "${job}" == "shell-unit-tests" ]]; then
            vote_run=true
        fi
    fi

    # Rule: ui-sources - UI source changes need UI tests
    if any_file_matches "${changed_files}" "^ui/.*\.(tsx?|jsx?|css|scss)$"; then
        if [[ "${job}" == "ui" ]]; then
            vote_run=true
        fi
        if [[ "${job}" == "ui-component" ]]; then
            vote_run=true
        fi
    fi

    # Rule: style-always - Style checks run on every PR
    if [[ -n "${changed_files}" ]]; then
        if [[ "${job}" == "style-check" ]]; then
            vote_run=true
        fi
    fi


    # Clash resolution: run wins (false positives better than false negatives)
    if [[ "${vote_run}" == "true" ]]; then
        if [[ "${vote_skip}" == "true" ]]; then
            echo "Info: Rule clash for ${job}: run and skip both voted, running (run wins)" >&2
        fi
        return 0
    fi

    if [[ "${vote_skip}" == "true" ]]; then
        return 1
    fi

    # No rules matched, check default
    case "${job}" in
        check-crd-compatibility)
            return 1  # default: skip
            ;;
        check-generated-files)
            return 0  # default: run
            ;;
        db-integration-tests)
            return 1  # default: skip
            ;;
        gke-nongroovy-e2e-tests)
            return 0  # default: run
            ;;
        gke-qa-e2e-tests)
            return 1  # default: skip
            ;;
        gke-upgrade-tests)
            return 1  # default: skip
            ;;
        go)
            return 0  # default: run
            ;;
        go-postgres)
            return 0  # default: run
            ;;
        golangci-lint)
            return 0  # default: run
            ;;
        ocp-qa-e2e-tests)
            return 1  # default: skip
            ;;
        sensor-integration-tests)
            return 0  # default: run
            ;;
        sensor-integration-tests-openshift)
            return 1  # default: skip
            ;;
        shell-unit-tests)
            return 0  # default: run
            ;;
        should-dispatch)
            return 0  # default: run
            ;;
        style-check)
            return 0  # default: run
            ;;
        ui)
            return 0  # default: run
            ;;
        ui-component)
            return 0  # default: run
            ;;
        wait-for-images)
            return 0  # default: run
            ;;
        *)
            echo "Warning: Unknown job ${job}, running by default" >&2
            return 0
            ;;
    esac
}
