package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

const prowGateTemplate = `#!/usr/bin/env bash
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

{{ range .RunAllTests.When -}}
{{- if .Label }}
    # Check label: {{ .Label }}
    if has_label "{{ .Label }}" "${pr_labels}"; then
        return 0
    fi
{{ end -}}
{{- if .Files }}
    # Check files match: {{ formatPatterns .Files }}
    while IFS= read -r file; do
        if file_matches_any "${file}" {{ formatBashArray .Files }}; then
            return 0
        fi
    done <<< "${changed_files}"
{{ end -}}
{{- end }}
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
{{- range $jobName, $job := .Jobs }}
        {{ $jobName }})
            # Job: {{ $jobName }}
            while IFS= read -r file; do
                local matches_when=false
                local matches_when_not=false

                # Check when conditions
{{- range $job.When }}
{{- if .Always }}
                # Always run
                matches_when=true
{{- end }}
{{- if .Label }}
                # Label: {{ .Label }}
                if has_label "{{ .Label }}" "${pr_labels}"; then
                    matches_when=true
                fi
{{- end }}
{{- if .Files }}
                # Files: {{ formatPatterns .Files }}
                if file_matches_any "${file}" {{ formatBashArray .Files }}; then
                    matches_when=true
                fi
{{- end }}
{{- end }}

                # If file matches when, check when_not
                if [[ "${matches_when}" == "true" ]]; then
{{- if $job.WhenNot }}
                    # Check when_not conditions
{{- range $job.WhenNot }}
{{- if .Label }}
                    # Label: {{ .Label }}
                    if has_label "{{ .Label }}" "${pr_labels}"; then
                        matches_when_not=true
                    fi
{{- end }}
{{- if .Files }}
                    # Files: {{ formatPatterns .Files }}
                    if file_matches_any "${file}" {{ formatBashArray .Files }}; then
                        matches_when_not=true
                    fi
{{- end }}
{{- end }}
{{- end }}

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
{{- end }}
        *)
            echo "Warning: Unknown job ${job}, running by default" >&2
            return 0
            ;;
    esac
}
`

func generateProwGateScript(config *Config, outputPath string) error {
	// Template functions
	funcMap := template.FuncMap{
		"formatBashArray": formatBashArray,
		"formatPatterns": func(patterns []string) string {
			return strings.Join(patterns, ", ")
		},
	}

	tmpl, err := template.New("prow").Funcs(funcMap).Parse(prowGateTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	// Make file executable
	if err := f.Chmod(0755); err != nil {
		return fmt.Errorf("setting executable: %w", err)
	}

	if err := tmpl.Execute(f, config); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}

func formatBashArray(patterns []string) string {
	quoted := make([]string, len(patterns))
	for i, p := range patterns {
		// Escape quotes for bash
		escaped := strings.ReplaceAll(p, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, `$`, `\$`)
		quoted[i] = fmt.Sprintf(`"%s"`, escaped)
	}
	return strings.Join(quoted, " ")
}
