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
    comments="$(gh api "repos/${repo}/pulls/${PULL_NUMBER}/comments" \
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
    grep -qF "${label}" <<< "${labels}"
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
{{ range .Rules }}
    # Rule: {{ .Name }}{{ if .Description }} - {{ .Description }}{{ end }}
    if {{ .Condition }}; then
        {{- range .RunJobs }}
        if [[ "${job}" == "{{ . }}" ]]; then
            vote_run=true
        fi
        {{- end }}
        {{- range .SkipJobs }}
        if [[ "${job}" == "{{ . }}" ]]; then
            vote_skip=true
        fi
        {{- end }}
    fi
{{ end }}

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
{{- range $job, $action := .Defaults }}
        {{ $job }})
            {{- if eq $action "run" }}
            return 0  # default: run
            {{- else }}
            return 1  # default: skip
            {{- end }}
            ;;
{{- end }}
        *)
            echo "Warning: Unknown job ${job}, running by default" >&2
            return 0
            ;;
    esac
}
`

type ProwRule struct {
	Name        string
	Description string
	Condition   string   // Bash condition
	RunJobs     []string // Job names to run
	SkipJobs    []string // Job names to skip
}

func generateProwGateScript(config *Config, outputPath string) error {
	// Convert rules to Prow format
	prowRules := convertToProwRules(config)

	tmpl, err := template.New("prow").Parse(prowGateTemplate)
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

	data := struct {
		Rules    []ProwRule
		Defaults map[string]string
	}{
		Rules:    prowRules,
		Defaults: config.Defaults,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}

func convertToProwRules(config *Config) []ProwRule {
	var prowRules []ProwRule

	for _, rule := range config.Rules {
		prowRule := ProwRule{
			Name:        rule.Name,
			Description: rule.Description,
		}

		// Build condition
		if rule.When.Label != "" {
			prowRule.Condition = fmt.Sprintf(`has_label "%s" "${pr_labels}"`, rule.When.Label)
		} else if len(rule.When.AnyFileMatches) > 0 {
			patterns := formatBashArray(rule.When.AnyFileMatches)
			prowRule.Condition = fmt.Sprintf(`any_file_matches "${changed_files}" %s`, patterns)
		} else if len(rule.When.EveryFileMatches) > 0 {
			patterns := formatBashArray(rule.When.EveryFileMatches)
			prowRule.Condition = fmt.Sprintf(`every_file_matches "${changed_files}" %s`, patterns)
		} else if rule.When.AnyFileChanged {
			prowRule.Condition = `[[ -n "${changed_files}" ]]`
		} else {
			prowRule.Condition = "false" // Should not happen due to validation
		}

		// Extract jobs to run/skip
		prowRule.RunJobs = expandJobList(rule.Action.Run, config.Jobs)
		prowRule.SkipJobs = expandJobList(rule.Action.Skip, config.Jobs)

		// Remove exceptions from skip list
		if len(rule.Action.Except) > 0 {
			prowRule.SkipJobs = removeJobs(prowRule.SkipJobs, rule.Action.Except)
		}

		prowRules = append(prowRules, prowRule)
	}

	return prowRules
}

func expandJobList(jobsInterface interface{}, allJobs []string) []string {
	if jobsInterface == nil {
		return nil
	}

	switch v := jobsInterface.(type) {
	case string:
		if v == "all" {
			return allJobs
		}
		return []string{v}
	case []interface{}:
		var jobs []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				if s == "all" {
					return allJobs
				}
				jobs = append(jobs, s)
			}
		}
		return jobs
	default:
		return nil
	}
}

func removeJobs(jobs, toRemove []string) []string {
	removeSet := make(map[string]bool)
	for _, job := range toRemove {
		removeSet[job] = true
	}

	var result []string
	for _, job := range jobs {
		if !removeSet[job] {
			result = append(result, job)
		}
	}
	return result
}

func formatBashArray(patterns []string) string {
	quoted := make([]string, len(patterns))
	for i, p := range patterns {
		// Escape quotes for bash
		escaped := strings.ReplaceAll(p, `"`, `\"`)
		quoted[i] = fmt.Sprintf(`"%s"`, escaped)
	}
	return strings.Join(quoted, " ")
}
