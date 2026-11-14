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
{{- range $jobName, $job := .Jobs }}
      should_run_{{ sanitizeName $jobName }}:
        description: >
          'true' if {{ $jobName }} should run based on test selection rules
        value: {{ "${{ jobs.detect.outputs.should_run_" }}{{ sanitizeName $jobName }}{{ " }}" }}
{{- end }}

jobs:
  detect:
    runs-on: ubuntu-24.04
    outputs:
      files: {{ "${{ steps.list.outputs.files }}" }}
      ui_only: {{ "${{ steps.list.outputs.ui_only }}" }}
{{- range $jobName, $job := .Jobs }}
      should_run_{{ sanitizeName $jobName }}: {{ "${{ steps.evaluate.outputs.should_run_" }}{{ sanitizeName $jobName }}{{ " }}" }}
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

          if [[ ${#files[@]} -gt 0 ]]; then
            {
              echo "files<<EOF"
              printf '%s\n' "${files[@]}"
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
              print("No files changed or not a PR, running all jobs by default", file=sys.stderr)
              sys.exit(0)

          files = [line.strip() for line in files_output.strip().split('\n') if line]
          print(f"Evaluating {len(files)} changed files", file=sys.stderr)

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
                  print(f"PR labels: {labels}", file=sys.stderr)
              except subprocess.CalledProcessError:
                  print("Warning: Could not fetch PR labels", file=sys.stderr)

          # Check for /test override comments
          test_overrides = set()
          if pr_number:
              try:
                  result = subprocess.run(
                      ['gh', 'pr', 'view', pr_number, '--repo', os.environ['GH_REPO'],
                       '--json', 'comments', '--jq', '.comments[].body'],
                      check=True, capture_output=True, text=True
                  )
                  for line in result.stdout.strip().split('\n'):
                      if line.startswith('/test '):
                          job = line[6:].strip()
                          test_overrides.add(job)
                          print(f"Found /test override: {job}", file=sys.stderr)
              except subprocess.CalledProcessError:
                  pass

          # Helper: Check if file matches any pattern
          def matches_any_pattern(file, patterns):
              for pattern in patterns:
                  try:
                      if re.search(pattern, file):
                          return True
                  except re.error as e:
                      print(f"Warning: Invalid regex '{pattern}': {e}", file=sys.stderr)
              return False

          # Helper: Check if condition matches (always, label, or files)
          def matches_condition(cond, file=None):
              if cond.get('always'):
                  return True
              if cond.get('label'):
                  return cond['label'] in labels
              if file and cond.get('files'):
                  return matches_any_pattern(file, cond['files'])
              return False

          # Check run_all_tests global trigger
          run_all = False
          for cond in {{ toPythonList .RunAllTests.When }}:
              if cond.get('label') and cond['label'] in labels:
                  run_all = True
                  print(f"run_all_tests triggered by label: {cond['label']}", file=sys.stderr)
                  break
              if cond.get('files'):
                  for file in files:
                      if matches_any_pattern(file, cond['files']):
                          run_all = True
                          print(f"run_all_tests triggered by file: {file}", file=sys.stderr)
                          break
              if run_all:
                  break

          # Evaluate each job with per-file logic
          with open(os.environ['GITHUB_OUTPUT'], 'a') as output:
{{- range $jobName, $job := .Jobs }}
              # Job: {{ $jobName }}
              should_run = False

              # Check /test override first (highest priority)
              if '{{ $jobName }}' in test_overrides:
                  should_run = True
                  print("{{ $jobName }}: RUN (/test override)", file=sys.stderr)
              # Check run_all_tests
              elif run_all:
                  should_run = True
                  print("{{ $jobName }}: RUN (run_all_tests)", file=sys.stderr)
              else:
                  # Per-file evaluation: run if ANY file matches when AND NOT when_not
                  when_conditions = {{ toPythonList $job.When }}
                  when_not_conditions = {{ toPythonList $job.WhenNot }}

                  for file in files:
                      # Check if file matches ANY when condition
                      matches_when = False
                      for cond in when_conditions:
                          if matches_condition(cond, file):
                              matches_when = True
                              break

                      if matches_when:
                          # Check if file matches ANY when_not condition
                          matches_when_not = False
                          for cond in when_not_conditions:
                              if matches_condition(cond, file):
                                  matches_when_not = True
                                  break

                          if not matches_when_not:
                              # File matches when AND NOT when_not → trigger!
                              should_run = True
                              print(f"{{ $jobName }}: RUN (triggered by {file})", file=sys.stderr)
                              break

                  if not should_run:
                      print("{{ $jobName }}: SKIP (no triggering files)", file=sys.stderr)

              output.write(f"should_run_{{ sanitizeName $jobName }}={str(should_run).lower()}\n")
{{- end }}
`

func generateGitHubActionsWorkflow(config *Config, outputPath string) error {
	// Template functions
	funcMap := template.FuncMap{
		"sanitizeName": sanitizeName,
		"toPythonList": toPythonList,
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

	if err := tmpl.Execute(f, config); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	return nil
}

// toPythonList converts a slice of WhenCondition to a Python list of dicts
func toPythonList(conditions []WhenCondition) string {
	if len(conditions) == 0 {
		return "[]"
	}

	var parts []string
	for _, cond := range conditions {
		dict := "{"
		fields := []string{}

		if cond.Always {
			fields = append(fields, "'always': True")
		}
		if cond.Label != "" {
			// Escape single quotes in label
			escaped := strings.ReplaceAll(cond.Label, "'", "\\'")
			fields = append(fields, fmt.Sprintf("'label': '%s'", escaped))
		}
		if len(cond.Files) > 0 {
			fileStrs := make([]string, len(cond.Files))
			for i, f := range cond.Files {
				// Escape single quotes and backslashes in patterns
				escaped := strings.ReplaceAll(f, "\\", "\\\\")
				escaped = strings.ReplaceAll(escaped, "'", "\\'")
				fileStrs[i] = fmt.Sprintf("'%s'", escaped)
			}
			fields = append(fields, fmt.Sprintf("'files': [%s]", strings.Join(fileStrs, ", ")))
		}

		dict += strings.Join(fields, ", ") + "}"
		parts = append(parts, dict)
	}

	return "[" + strings.Join(parts, ", ") + "]"
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
