package main

import (
	"fmt"
	"os"
	"strings"
	"text/template"
)

const ghaWorkflowTemplate = `# GENERATED FILE - DO NOT EDIT
# Generated from: ci/test-selection-rules.yaml
# To regenerate: make generate-ci-selection

name: Detect changed files

on:
  workflow_call:
    inputs:
      run_all:
        description: >
          Newline-separated list of patterns (same ^/$ anchored
          format as the output). If any changed file matches any
          of these, output is empty and all jobs run.
        type: string
        required: false
        default: ''
    outputs:
      files:
        description: >
          Newline-separated list of changed filenames, or empty
          string on push/failure (falsy, so jobs default to run).
        value: {{ "${{ jobs.detect.outputs.files }}" }}
      ui_only:
        description: >
          'true' when every changed file is under ui/, empty
          otherwise. Skip non-UI jobs with: if: !needs.X.outputs.ui_only
        value: {{ "${{ jobs.detect.outputs.ui_only }}" }}
{{- range .DomainFlags }}
      {{ .Name }}:
        description: >
          {{ .Description }}
        value: {{ "${{ jobs.detect.outputs." }}{{ .Name }}{{ " }}" }}
{{- end }}

jobs:
  detect:
    runs-on: ubuntu-24.04
    outputs:
      files: {{ "${{ steps.list.outputs.files }}" }}
      ui_only: {{ "${{ steps.list.outputs.ui_only }}" }}
{{- range .DomainFlags }}
      {{ .Name }}: {{ "${{ steps.evaluate.outputs." }}{{ .Name }}{{ " }}" }}
{{- end }}
    steps:
      - name: Checkout
        uses: actions/checkout@9c091bb21b7c1c1d1991bb908d89e4e9dddfe3e0
        with:
          fetch-depth: 0

      - name: List changed files
        id: list
        env:
          GH_TOKEN: {{ "${{ github.token }}" }}
          GH_REPO: {{ "${{ github.repository }}" }}
          PR_NUMBER: {{ "${{ github.event.pull_request.number }}" }}
          RUN_ALL: {{ "${{ inputs.run_all }}" }}
          WORKFLOW_FILE_PATH: ".github/workflows/detect-changes-generated.yml"
        shell: bash
        run: |
          set -eo pipefail

          if [[ -z "${PR_NUMBER}" ]]; then
            echo "Not a PR" >&2
            exit 0
          fi

          if ! output="$(gh api "repos/${GH_REPO}/pulls/${PR_NUMBER}/files" --paginate --jq '.[].filename')"; then
            echo "::warning::GitHub API call failed" >&2
            exit 0
          fi

          mapfile -t files < <(echo -n "${output}")
          echo "Changed files (${#files[@]}):" >&2
          printf '  %s\n' "${files[@]}" >&2

          if [[ ${#files[@]} -ge 3000 ]]; then
            echo "::warning::PR has 3000+ changed files" >&2
            exit 0
          fi

          anchored=$(printf '^%s$\n' "${files[@]}")

          if echo "${anchored}" | grep -qF "^${WORKFLOW_FILE_PATH}$"; then
            echo "detect-changes-generated.yml modified, running all jobs" >&2
            exit 0
          fi

          if [[ -n "${RUN_ALL}" ]]; then
            while IFS= read -r pattern; do
              [[ -z "$pattern" ]] && continue
              if echo "${anchored}" | grep -qF "$pattern"; then
                echo "run_all: matched '${pattern}', running all jobs" >&2
                exit 0
              fi
            done <<< "${RUN_ALL}"
          fi

          if [[ ${#files[@]} -gt 0 ]]; then
            {
              echo "files<<EOF"
              printf '^%s$\n' "${files[@]}"
              echo "EOF"
            } >> "${GITHUB_OUTPUT}"

            if ! echo "${anchored}" | grep -qv '^\^ui/'; then
              echo "All changed files are under ui/" >&2
              echo "ui_only=true" >> "${GITHUB_OUTPUT}"
            fi
          fi

      - name: Evaluate test selection rules
        id: evaluate
        shell: python
        env:
          GH_TOKEN: {{ "${{ github.token }}" }}
          GH_REPO: {{ "${{ github.repository }}" }}
          PR_NUMBER: {{ "${{ github.event.pull_request.number }}" }}
          FILES: {{ "${{ steps.list.outputs.files }}" }}
        run: |
          import os
          import re
          import subprocess
          import sys

          pr_number = os.environ.get('PR_NUMBER')
          files_output = os.environ.get('FILES', '')

          # Get changed files
          if not files_output:
              print("No files changed or not a PR, skipping rule evaluation", file=sys.stderr)
              sys.exit(0)

          files = [line.strip('^$') for line in files_output.strip().split('\n') if line]

          # Get PR labels
          labels = []
          if pr_number:
              try:
                  result = subprocess.run(
                      ['gh', 'pr', 'view', pr_number, '--repo', os.environ['GH_REPO'],
                       '--json', 'labels', '--jq', '.labels[].name'],
                      check=True, capture_output=True, text=True
                  )
                  labels = result.stdout.strip().split('\n') if result.stdout.strip() else []
              except subprocess.CalledProcessError:
                  print("Warning: Could not fetch PR labels", file=sys.stderr)

          print(f"Files: {len(files)}", file=sys.stderr)
          print(f"Labels: {labels}", file=sys.stderr)

          # Helper functions
          def any_file_matches(patterns):
              for pattern in patterns:
                  regex = re.compile(pattern)
                  if any(regex.search(f) for f in files):
                      return True
              return False

          def every_file_matches(patterns):
              if not files:
                  return False
              for f in files:
                  if not any(re.search(pattern, f) for pattern in patterns):
                      return False
              return True

          def has_label(label):
              return label in labels

          # Evaluate rules and output domain flags
          with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
{{- range .DomainFlags }}
              # {{ .Name }}: {{ .Description }}
              {{ .Name }}_value = (
                  {{- range $i, $cond := .Conditions }}
                  {{- if $i }} or {{ end -}}
                  ({{ $cond }})
                  {{- end }}
              )
              print(f"{{ .Name }}={str({{ .Name }}_value).lower()}", file=sys.stderr)
              output.write(f"{{ .Name }}={str({{ .Name }}_value).lower()}\n")
{{ end }}
`

type DomainFlag struct {
	Name        string
	Description string
	Conditions  []string // Python conditions
}

func generateGitHubActionsWorkflow(config *Config, outputPath string) error {
	// Extract domain flags from rules
	domainFlags := extractDomainFlags(config)

	tmpl, err := template.New("workflow").Parse(ghaWorkflowTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	data := struct {
		DomainFlags []DomainFlag
	}{
		DomainFlags: domainFlags,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}

func extractDomainFlags(config *Config) []DomainFlag {
	// Create domain flags for job groups and label triggers
	var flags []DomainFlag

	// Add flags for common domains based on rules
	// Track insertion order for deterministic output
	domainMap := make(map[string]*DomainFlag)
	domainOrder := []string{}

	for _, rule := range config.Rules {
		// Extract domain from rule
		var domain string
		var conditions []string

		// Label-based rules
		if rule.When.Label != "" {
			domain = sanitizeName(rule.When.Label + "_label")
			conditions = append(conditions, fmt.Sprintf("has_label(%q)", rule.When.Label))
		}

		// Path-based rules for subsystems
		if len(rule.When.AnyFileMatches) > 0 {
			// Try to extract domain name from patterns
			domain = sanitizeName(rule.Name)
			patterns := formatPatterns(rule.When.AnyFileMatches)
			conditions = append(conditions, fmt.Sprintf("any_file_matches([%s])", patterns))
		}

		if len(rule.When.EveryFileMatches) > 0 {
			domain = sanitizeName(rule.Name)
			patterns := formatPatterns(rule.When.EveryFileMatches)
			conditions = append(conditions, fmt.Sprintf("every_file_matches([%s])", patterns))
		}

		if rule.When.AnyFileChanged {
			domain = sanitizeName(rule.Name)
			conditions = append(conditions, "bool(files)")
		}

		if domain == "" || len(conditions) == 0 {
			continue
		}

		// Merge with existing domain flag if present
		if existing, ok := domainMap[domain]; ok {
			existing.Conditions = append(existing.Conditions, conditions...)
		} else {
			domainMap[domain] = &DomainFlag{
				Name:        domain,
				Description: fmt.Sprintf("'true' if %s", rule.Description),
				Conditions:  conditions,
			}
			domainOrder = append(domainOrder, domain)
		}
	}

	// Convert map to slice in insertion order (deterministic)
	for _, domain := range domainOrder {
		flags = append(flags, *domainMap[domain])
	}

	return flags
}

func sanitizeName(name string) string {
	// Convert to valid identifier: lowercase, replace non-alphanumeric with underscore
	result := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + ('a' - 'A') // lowercase
		}
		return '_'
	}, name)

	// Remove consecutive underscores and leading/trailing underscores
	result = strings.Trim(result, "_")
	for strings.Contains(result, "__") {
		result = strings.ReplaceAll(result, "__", "_")
	}

	return result
}

func formatPatterns(patterns []string) string {
	quoted := make([]string, len(patterns))
	for i, p := range patterns {
		// Escape quotes in pattern
		escaped := strings.ReplaceAll(p, `"`, `\"`)
		quoted[i] = fmt.Sprintf(`"%s"`, escaped)
	}
	return strings.Join(quoted, ", ")
}
