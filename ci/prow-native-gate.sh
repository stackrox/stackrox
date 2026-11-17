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

# Check if file matches any of the patterns (extended regex)
file_matches_any() {
    local file="$1"
    shift
    local patterns=("$@")

    for pattern in "${patterns[@]}"; do
        if grep -qE "${pattern}" <<< "${file}"; then
            return 0
        fi
    done
    return 1
}

# Check run_all_tests conditions
check_run_all_tests() {
    local changed_files="$1"
    local pr_labels="$2"


    # Check label: ci-run-all-tests
    if has_label "ci-run-all-tests" "${pr_labels}"; then
        return 0
    fi

    # Check files match: ^go\.mod$, ^go\.sum$, ^proto/, ^generated/, \.proto$
    while IFS= read -r file; do
        if file_matches_any "${file}" "^go\.mod\$" "^go\.sum\$" "^proto/" "^generated/" "\.proto\$"; then
            return 0
        fi
    done <<< "${changed_files}"

    return 1
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
        pr_labels=""
    fi

    # Check for /test override first (highest priority)
    if has_test_override "${job}"; then
        echo "Info: /test override detected for ${job}" >&2
        return 0
    fi

    # Check run_all_tests global trigger
    if check_run_all_tests "${changed_files}" "${pr_labels}"; then
        echo "Info: run_all_tests triggered for ${job}" >&2
        return 0
    fi

    # Per-job evaluation with per-file logic
    case "${job}" in
        check-crd-compatibility)
            # Job: check-crd-compatibility
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Label: area/operator
                if has_label "area/operator" "${pr_labels}"; then
                    matches_when=true
                fi
                # Files: ^operator/, \.crd\.yaml$, ^config-controller/
                if file_matches_any "${file}" "^operator/" "\.crd\.yaml\$" "^config-controller/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        check-generated-files)
            # Job: check-generated-files
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: ^proto/, ^generated/
                if file_matches_any "${file}" "^proto/" "^generated/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        db-integration-tests)
            # Job: db-integration-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Label: area/scanner
                if has_label "area/scanner" "${pr_labels}"; then
                    matches_when=true
                fi
                # Files: ^scanner/
                if file_matches_any "${file}" "^scanner/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        e2e-byodb-tests)
            # Job: e2e-byodb-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: ^central/, ^scanner/
                if file_matches_any "${file}" "^central/" "^scanner/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^ui/, ^roxctl/, ^operator/, ^sensor/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^ui/" "^roxctl/" "^operator/" "^sensor/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        e2e-db-backup-restore-tests)
            # Job: e2e-db-backup-restore-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: ^central/
                if file_matches_any "${file}" "^central/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^ui/, ^roxctl/, ^operator/, ^sensor/, ^scanner/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^ui/" "^roxctl/" "^operator/" "^sensor/" "^scanner/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        e2e-gke-upgrade-tests)
            # Job: e2e-gke-upgrade-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: ^central/, ^sensor/, ^scanner/, ^operator/
                if file_matches_any "${file}" "^central/" "^sensor/" "^scanner/" "^operator/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^ui/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^ui/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        e2e-nongroovy-tests)
            # Job: e2e-nongroovy-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: ^roxctl/, ^central/, ^sensor/
                if file_matches_any "${file}" "^roxctl/" "^central/" "^sensor/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^ui/, ^operator/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^ui/" "^operator/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        e2e-qa-tests-gke)
            # Job: e2e-qa-tests-gke
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        e2e-qa-tests-gke-konflux)
            # Job: e2e-qa-tests-gke-konflux
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        e2e-qa-tests-ocp-4-22)
            # Job: e2e-qa-tests-ocp-4-22
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        github-actions-lint)
            # Job: github-actions-lint
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: ^\.github/workflows/
                if file_matches_any "${file}" "^\.github/workflows/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        gke-nongroovy-e2e-tests)
            # Job: gke-nongroovy-e2e-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: ^roxctl/, ^central/, ^sensor/
                if file_matches_any "${file}" "^roxctl/" "^central/" "^sensor/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^ui/, ^operator/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^ui/" "^operator/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        gke-qa-e2e-tests)
            # Job: gke-qa-e2e-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        gke-ui-e2e-tests)
            # Job: gke-ui-e2e-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Label: area/ui
                if has_label "area/ui" "${pr_labels}"; then
                    matches_when=true
                fi
                # Files: ^ui/
                if file_matches_any "${file}" "^ui/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        go)
            # Job: go
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: \.go$
                if file_matches_any "${file}" "\.go\$"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^ui/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^ui/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        go-postgres)
            # Job: go-postgres
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Label: area/central
                if has_label "area/central" "${pr_labels}"; then
                    matches_when=true
                fi
                # Files: ^central/(sac|search|auth|authprovider|apitoken|user|group|role|rbac|jwt|serviceidentities)/, ^central/(postgres|globaldb|globalindex|graphdb|dackbox|blob|views)/, ^central/(notifier|notifiers|reports|reportconfigurations)/, ^central/(externalbackups|declarativeconfig|config|installation)/, ^central/(ui|graphql|grpc|endpoints)/, ^central/(featureflags|development|audit|logimbue|trace)/, ^central/(helmcharts|ping|version|systeminfo|docs|internal)/, ^pkg/(sac|search|postgres|notifiers|migrations|renderer|backup)/, ^central/(cluster|deployment|image|baseimage|node|pod|namespace|secret|serviceaccount|virtualmachine)/, ^central/(alert|detection|policy|policycat)/, ^central/(networkbaseline|networkgraph|networkpolicies)/, ^central/(complianceoperator|processindicator)/, ^central/(imageintegration|delegatedregistryconfig|enrichment|reprocessor)/, ^central/(sensor|sensorupgrade|clusterinit|securedclustercertgen)/, ^central/(metrics|telemetry|debug|scrape|vulnmgmt|hash)/, ^pkg/(grpc|sync|concurrency|features|logging|utils|set|retry|env|errorhelpers|testutils|mtls|x509utils|buildinfo|version|cryptoutils|urlfmt|stringutils|timestamps|cloudproviders|cloudsources|roxctl)/, ^pkg/(centralsensor|kubernetes|detection|networkgraph|protoconv|sensor|alerts|images|metrics|telemetry)/
                if file_matches_any "${file}" "^central/(sac|search|auth|authprovider|apitoken|user|group|role|rbac|jwt|serviceidentities)/" "^central/(postgres|globaldb|globalindex|graphdb|dackbox|blob|views)/" "^central/(notifier|notifiers|reports|reportconfigurations)/" "^central/(externalbackups|declarativeconfig|config|installation)/" "^central/(ui|graphql|grpc|endpoints)/" "^central/(featureflags|development|audit|logimbue|trace)/" "^central/(helmcharts|ping|version|systeminfo|docs|internal)/" "^pkg/(sac|search|postgres|notifiers|migrations|renderer|backup)/" "^central/(cluster|deployment|image|baseimage|node|pod|namespace|secret|serviceaccount|virtualmachine)/" "^central/(alert|detection|policy|policycat)/" "^central/(networkbaseline|networkgraph|networkpolicies)/" "^central/(complianceoperator|processindicator)/" "^central/(imageintegration|delegatedregistryconfig|enrichment|reprocessor)/" "^central/(sensor|sensorupgrade|clusterinit|securedclustercertgen)/" "^central/(metrics|telemetry|debug|scrape|vulnmgmt|hash)/" "^pkg/(grpc|sync|concurrency|features|logging|utils|set|retry|env|errorhelpers|testutils|mtls|x509utils|buildinfo|version|cryptoutils|urlfmt|stringutils|timestamps|cloudproviders|cloudsources|roxctl)/" "^pkg/(centralsensor|kubernetes|detection|networkgraph|protoconv|sensor|alerts|images|metrics|telemetry)/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^ui/, ^scanner/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^ui/" "^scanner/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        golangci-lint)
            # Job: golangci-lint
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: \.go$, ^\.golangci\.yml$
                if file_matches_any "${file}" "\.go\$" "^\.golangci\.yml\$"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^ui/, ^docs/, \.md$, ^CHANGELOG, ^README, ^\.github/, ^\.openshift-ci/, ^scripts/ci/
                    if file_matches_any "${file}" "^ui/" "^docs/" "\.md\$" "^CHANGELOG" "^README" "^\.github/" "^\.openshift-ci/" "^scripts/ci/"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        local-roxctl-tests)
            # Job: local-roxctl-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Label: area/roxctl
                if has_label "area/roxctl" "${pr_labels}"; then
                    matches_when=true
                fi
                # Files: ^roxctl/
                if file_matches_any "${file}" "^roxctl/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        ocp-vm-scanning-e2e-tests)
            # Job: ocp-vm-scanning-e2e-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        openshift-ci-lint)
            # Job: openshift-ci-lint
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: ^\.openshift-ci/
                if file_matches_any "${file}" "^\.openshift-ci/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        sensor-integration-tests)
            # Job: sensor-integration-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Label: area/sensor
                if has_label "area/sensor" "${pr_labels}"; then
                    matches_when=true
                fi
                # Files: ^sensor/(admission-control|upgrader|debugger|testutils|tests)/, ^sensor/kubernetes/(fake|client|enforcer)/, ^pkg/(admissioncontrol|docker|enforcers|complianceoperator|k8sintrospect|protocompat|compliance/checks|binenc)/, ^sensor/common/(admissioncontroller|centralclient|centralproxy|compliance|config|detector|enforcer|networkflow|sensor|image|scannerclient|virtualmachine)/, ^sensor/kubernetes/(eventpipeline|listener|complianceoperator|networkpolicies|telemetry|upgrade|clusterhealth|clustermetrics|clusterstatus|certrefresh)/, ^pkg/(grpc|sync|concurrency|features|logging|utils|set|retry|env|errorhelpers|testutils|mtls|x509utils|buildinfo|version|cryptoutils|urlfmt|stringutils|timestamps|cloudproviders|cloudsources|roxctl)/, ^pkg/(centralsensor|kubernetes|detection|networkgraph|protoconv|sensor|alerts|images|metrics|telemetry)/
                if file_matches_any "${file}" "^sensor/(admission-control|upgrader|debugger|testutils|tests)/" "^sensor/kubernetes/(fake|client|enforcer)/" "^pkg/(admissioncontrol|docker|enforcers|complianceoperator|k8sintrospect|protocompat|compliance/checks|binenc)/" "^sensor/common/(admissioncontroller|centralclient|centralproxy|compliance|config|detector|enforcer|networkflow|sensor|image|scannerclient|virtualmachine)/" "^sensor/kubernetes/(eventpipeline|listener|complianceoperator|networkpolicies|telemetry|upgrade|clusterhealth|clustermetrics|clusterstatus|certrefresh)/" "^pkg/(grpc|sync|concurrency|features|logging|utils|set|retry|env|errorhelpers|testutils|mtls|x509utils|buildinfo|version|cryptoutils|urlfmt|stringutils|timestamps|cloudproviders|cloudsources|roxctl)/" "^pkg/(centralsensor|kubernetes|detection|networkgraph|protoconv|sensor|alerts|images|metrics|telemetry)/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^ui/, ^scanner/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^ui/" "^scanner/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        shell-unit-tests)
            # Job: shell-unit-tests
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Files: \.sh$, ^scripts/, ^tests/.*\.sh$
                if file_matches_any "${file}" "\.sh\$" "^scripts/" "^tests/.*\.sh\$"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        should-dispatch)
            # Job: should-dispatch
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Always run
                matches_when=true

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        style-check)
            # Job: style-check
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Always run
                matches_when=true

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        ui)
            # Job: ui
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Label: area/ui
                if has_label "area/ui" "${pr_labels}"; then
                    matches_when=true
                fi
                # Files: ^ui/
                if file_matches_any "${file}" "^ui/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^central/, ^sensor/, ^scanner/, ^operator/, ^roxctl/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^central/" "^sensor/" "^scanner/" "^operator/" "^roxctl/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        ui-component)
            # Job: ui-component
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Label: area/ui
                if has_label "area/ui" "${pr_labels}"; then
                    matches_when=true
                fi
                # Files: ^ui/
                if file_matches_any "${file}" "^ui/"; then
                    matches_when=true
                fi

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
                    # Check when_not conditions
                    # Files: ^central/, ^sensor/, ^scanner/, \.md$, ^docs/, ^CHANGELOG, ^README
                    if file_matches_any "${file}" "^central/" "^sensor/" "^scanner/" "\.md\$" "^docs/" "^CHANGELOG" "^README"; then
                        matches_when_not=true
                    fi

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        wait-for-images)
            # Job: wait-for-images
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
                # Always run
                matches_when=true

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then

                    # If matches when AND NOT when_not, job should run
                    if [[ "${matches_when_not}" == "false" ]]; then
                        echo "Info: ${job} triggered by ${file}" >&2
                        return 0
                    fi
                fi
            done <<< "${changed_files}"

            echo "Info: ${job} skipped (no triggering files)" >&2
            return 1
            ;;
        *)
            echo "Warning: Unknown job ${job}, running by default" >&2
            return 0
            ;;
    esac
}
