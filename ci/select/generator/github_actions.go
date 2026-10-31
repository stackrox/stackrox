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
{{- range .Jobs }}
      should_run_{{ .SanitizedName }}:
        description: >
          'true' if {{ .Name }} should run based on test selection rules
        value: {{ "${{ jobs.detect.outputs.should_run_" }}{{ .SanitizedName }}{{ " }}" }}
{{- end }}

jobs:
  detect:
    runs-on: ubuntu-24.04
    outputs:
      files: {{ "${{ steps.list.outputs.files }}" }}
      ui_only: {{ "${{ steps.list.outputs.ui_only }}" }}
{{- range .Jobs }}
      should_run_{{ .SanitizedName }}: {{ "${{ steps.evaluate.outputs.should_run_" }}{{ .SanitizedName }}{{ " }}" }}
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

          # Evaluate rules for each job and apply clash resolution
          job_decisions = {}
{{- range .Jobs }}
          job_decisions['{{ .Name }}'] = {'run': False, 'skip': False}
{{- end }}

          # Apply rules in order
{{- range .Rules }}
          # Rule: {{ .Name }} - {{ .Description }}
          rule_matches = False
{{- if .When.Label }}
          if has_label({{ .When.Label | printf "%q" }}):
              rule_matches = True
{{- end }}
{{- if .When.AnyFileMatches }}
          if any_file_matches([{{ range $i, $p := .When.AnyFileMatches }}{{ if $i }}, {{ end }}{{ $p | printf "%q" }}{{ end }}]):
              rule_matches = True
{{- end }}
{{- if .When.EveryFileMatches }}
          if every_file_matches([{{ range $i, $p := .When.EveryFileMatches }}{{ if $i }}, {{ end }}{{ $p | printf "%q" }}{{ end }}]):
              rule_matches = True
{{- end }}
{{- if .When.AnyFileChanged }}
          if bool(files):
              rule_matches = True
{{- end }}

          if rule_matches:
              print(f"Rule matched: {{ .Name }}", file=sys.stderr)
{{- if isRunAll .Action.Run }}
              # Run all jobs
{{- range $.Jobs }}
              job_decisions['{{ .Name }}']['run'] = True
{{- end }}
{{- else if .Action.Run }}
              # Run specific jobs
{{- range asStringSlice .Action.Run }}
              job_decisions['{{ . }}']['run'] = True
{{- end }}
{{- end }}
{{- if isSkipAll .Action.Skip }}
              # Skip all jobs (except those in 'except' list)
{{- $except := .Action.Except }}
{{- range $.Jobs }}
{{- if not (has $except .Name) }}
              job_decisions['{{ .Name }}']['skip'] = True
{{- end }}
{{- end }}
{{- else if .Action.Skip }}
              # Skip specific jobs
{{- range asStringSlice .Action.Skip }}
              job_decisions['{{ . }}']['skip'] = True
{{- end }}
{{- end }}

{{- end }}

          # Apply clash resolution: run wins over skip
          # Output final decisions
          with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
{{- range .Jobs }}
              should_run = job_decisions['{{ .Name }}']['run'] or not job_decisions['{{ .Name }}']['skip']
              # Default from config
{{- if eq (index $.Defaults .Name) "run" }}
              if not job_decisions['{{ .Name }}']['run'] and not job_decisions['{{ .Name }}']['skip']:
                  should_run = True
{{- else }}
              if not job_decisions['{{ .Name }}']['run'] and not job_decisions['{{ .Name }}']['skip']:
                  should_run = False
{{- end }}
              # Clash resolution: if both run and skip voted, run wins
              if job_decisions['{{ .Name }}']['run']:
                  should_run = True

              print(f"{{ .Name }}: run={job_decisions['{{ .Name }}']['run']}, skip={job_decisions['{{ .Name }}']['skip']}, decision={'run' if should_run else 'skip'}", file=sys.stderr)
              output.write(f"should_run_{{ .SanitizedName }}={str(should_run).lower()}\n")
{{- end }}
`

type JobInfo struct {
	Name          string
	SanitizedName string
}

func generateGitHubActionsWorkflow(config *Config, outputPath string) error {
	// Prepare jobs with sanitized names
	jobs := make([]JobInfo, len(config.Jobs))
	for i, job := range config.Jobs {
		jobs[i] = JobInfo{
			Name:          job,
			SanitizedName: sanitizeName(job),
		}
	}

	// Template functions
	funcMap := template.FuncMap{
		"has": func(slice []string, item string) bool {
			for _, s := range slice {
				if s == item {
					return true
				}
			}
			return false
		},
		"index": func(m map[string]string, key string) string {
			return m[key]
		},
		"isRunAll": func(run interface{}) bool {
			if run == nil {
				return false
			}
			s, ok := run.(string)
			return ok && s == "all"
		},
		"isSkipAll": func(skip interface{}) bool {
			if skip == nil {
				return false
			}
			s, ok := skip.(string)
			return ok && s == "all"
		},
		"asStringSlice": func(v interface{}) []string {
			if v == nil {
				return nil
			}
			// Handle both []string and []interface{}
			switch val := v.(type) {
			case []string:
				return val
			case []interface{}:
				result := make([]string, len(val))
				for i, item := range val {
					if s, ok := item.(string); ok {
						result[i] = s
					}
				}
				return result
			}
			return nil
		},
	}

	tmpl, err := template.New("workflow").Funcs(funcMap).Parse(ghaWorkflowTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	data := struct {
		Jobs     []JobInfo
		Rules    []Rule
		Defaults map[string]string
	}{
		Jobs:     jobs,
		Rules:    config.Rules,
		Defaults: config.Defaults,
	}

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
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
