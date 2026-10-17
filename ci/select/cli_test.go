package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, path, text string, mode os.FileMode) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(text), mode))
}

func TestResolveCLI(t *testing.T) {
	root := t.TempDir()
	files, labels := filepath.Join(root, "files"), filepath.Join(root, "labels")
	writeTestFile(t, files, "README.md\n", 0o600)
	writeTestFile(t, labels, "", 0o600)
	decision, log := filepath.Join(root, "decision"), filepath.Join(root, "log")
	args := []string{"resolve", "--rules", "testdata/rules.toml", "--defaults", "testdata/defaults",
		"--files-file", files, "--labels-file", labels, "--decision-out", decision, "--log-out", log}
	code, err := run(t.Context(), args, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	require.NoError(t, err)
	assert.Zero(t, code)
	text, err := os.ReadFile(decision)
	require.NoError(t, err)
	assert.Equal(t, "style-check\nwait-for-images\nshould-dispatch\n", string(text))
	text, err = os.ReadFile(log)
	require.NoError(t, err)
	assert.Contains(t, string(text), "requires it")
	args[2] = "testdata/invalid/bad-version.toml"
	code, err = run(t.Context(), args, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	require.Error(t, err)
	assert.NotZero(t, code)
}

func TestResolveCLIEmptyDiffAndFailures(t *testing.T) {
	inputs := t.TempDir()
	files := filepath.Join(inputs, "files")
	labels := filepath.Join(inputs, "labels")
	writeTestFile(t, files, "", 0o600)
	writeTestFile(t, labels, "", 0o600)
	for name, test := range map[string]struct {
		rules, defaults, files string
		expected, errorText    string
	}{
		"empty known diff": {
			rules: "testdata/corners.toml", defaults: "testdata/corners-defaults", files: files, expected: "unit\n",
		},
		"invalid rules": {
			rules: "testdata/invalid/bad-version.toml", defaults: "testdata/corners-defaults", files: files,
			errorText: "unsupported rules version: 2",
		},
		"broken rules": {
			rules: "testdata/invalid/broken.toml", defaults: "testdata/corners-defaults", files: files,
			errorText: "read TOML rules",
		},
		"missing changed files": {
			rules: "testdata/corners.toml", defaults: "testdata/corners-defaults", files: filepath.Join(inputs, "missing"),
			errorText: "read changed files",
		},
		"invalid defaults": {
			rules: "testdata/corners.toml", defaults: fixturePath(t, "unit maybe\n"), files: files,
			errorText: "decision must be run or skip",
		},
		"missing defaults": {
			rules: "testdata/corners.toml", defaults: filepath.Join(inputs, "missing"), files: files,
			errorText: "read decision-defaults",
		},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			decision, log := filepath.Join(folder, "decision"), filepath.Join(folder, "log")
			var output, diagnostic bytes.Buffer
			code, err := run(t.Context(), []string{
				"resolve", "--rules", test.rules, "--defaults", test.defaults,
				"--files-file", test.files, "--labels-file", labels,
				"--decision-out", decision, "--log-out", log,
				"--repo", "stackrox/stackrox", "--pr", "23035", "--commit", "a1b2c3d",
			}, strings.NewReader(""), &output, &diagnostic)
			if test.errorText != "" {
				require.ErrorContains(t, err, test.errorText)
				assert.Equal(t, 1, code)
				_, err = os.Stat(decision)
				assert.ErrorIs(t, err, os.ErrNotExist)
				_, err = os.Stat(log)
				assert.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			require.NoError(t, err)
			assert.Zero(t, code)
			text, err := os.ReadFile(decision)
			require.NoError(t, err)
			assert.Equal(t, test.expected, string(text))
			text, err = os.ReadFile(log)
			require.NoError(t, err)
			assert.Contains(t, string(text), "# stackrox/stackrox PR 23035 a1b2c3d")
			assert.Contains(t, string(text), "nothing-changed")
			assert.Empty(t, output.String())
		})
	}
}

func TestResolveCLIProcess(t *testing.T) {
	root := t.TempDir()
	binary := filepath.Join(root, "ci-select")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", binary, ".")
	text, err := build.CombinedOutput()
	require.NoError(t, err, string(text))
	files := filepath.Join(root, "files")
	writeTestFile(t, files, "README.md\n", 0o600)
	for name, test := range map[string]struct {
		rules string
		fail  bool
	}{
		"success":       {rules: "testdata/rules.toml"},
		"invalid rules": {rules: "testdata/invalid/bad-version.toml", fail: true},
	} {
		t.Run(name, func(t *testing.T) {
			folder := t.TempDir()
			decision, log := filepath.Join(folder, "decision"), filepath.Join(folder, "log")
			var output, diagnostic bytes.Buffer
			command := exec.CommandContext(t.Context(), binary,
				"resolve", "--rules", test.rules, "--defaults", "testdata/defaults", "--files-file", files,
				"--decision-out", decision, "--log-out", log, "--pr", "23035")
			command.Stdout, command.Stderr = &output, &diagnostic
			err := command.Run()
			if test.fail {
				require.Error(t, err)
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit)
				assert.Equal(t, 1, exit.ExitCode())
				assert.Contains(t, diagnostic.String(), "ci-select:")
				assert.Contains(t, diagnostic.String(), "unsupported rules version: 2")
				_, err = os.Stat(decision)
				assert.ErrorIs(t, err, os.ErrNotExist)
				return
			}
			require.NoError(t, err, diagnostic.String())
			assert.Empty(t, diagnostic.String())
			text, err := os.ReadFile(decision)
			require.NoError(t, err)
			assert.Equal(t, "style-check\nwait-for-images\nshould-dispatch\n", string(text))
			text, err = os.ReadFile(log)
			require.NoError(t, err)
			assert.Contains(t, string(text), "PR 23035")
		})
	}
}

func TestCLIValidation(t *testing.T) {
	for name, args := range map[string][]string{
		"unknown command":   {"bad"},
		"unknown mode":      {"gha", "bad", "--files-unknown"},
		"missing job":       {"gha", "gate", "--files-unknown"},
		"both diff options": {"gha", "print", "--files-file", "file", "--files-unknown"},
		"missing diff":      {"gha", "print"}, "extra argument": {"validate", "extra"},
	} {
		t.Run(name, func(t *testing.T) {
			code, err := run(t.Context(), args, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
			require.Error(t, err)
			assert.NotZero(t, code)
		})
	}
}

func TestCLIHelp(t *testing.T) {
	commands := map[string][]string{
		"root": {}, "resolve": {"resolve"}, "validate": {"validate"},
		"gha": {"gha"}, "gha print": {"gha", "print"}, "gha gate": {"gha", "gate"},
		"prow": {"prow"}, "prow gate": {"prow", "gate"},
		"gha-ci": {"gha-ci"}, "gha-dispatch-ci": {"gha-dispatch-ci"}, "prow-ci": {"prow-ci"},
	}
	for name, command := range commands {
		for _, flag := range []string{"-h", "--help"} {
			t.Run(name+" "+flag, func(t *testing.T) {
				var output, diagnostic bytes.Buffer
				args := append(append([]string{}, command...), flag)
				code, err := run(t.Context(), args, strings.NewReader(""), &output, &diagnostic)
				require.NoError(t, err)
				assert.Zero(t, code)
				assert.Empty(t, diagnostic.String())
				assert.Contains(t, output.String(), "Usage:")
				assert.Contains(t, output.String(), "Flags:")
				assert.Contains(t, output.String(), "-h, --help")
			})
		}
	}
}

func TestCLIHelpContent(t *testing.T) {
	for name, test := range map[string]struct {
		args             []string
		contains, absent []string
	}{
		"no command shows overview": {
			contains: []string{"Available Commands:", "Examples:", "Select CI jobs", "gha-dispatch-ci", "prow-ci"},
		},
		"group shows subcommands": {args: []string{"gha"}, contains: []string{"gate", "print"}},
		"help command":            {args: []string{"help", "resolve"}, contains: []string{"--files-file", "--log-out", "Examples:"}},
		"validate flags":          {args: []string{"validate", "-h"}, contains: []string{"--rules", "--defaults"}, absent: []string{"--comments-file", "--enforce"}},
		"GHA gate flags":          {args: []string{"gha", "gate", "-h"}, contains: []string{"--job", "--enforce", "--files-unknown"}, absent: []string{"--comments-file"}},
		"Prow local help":         {args: []string{"prow", "gate", "-h"}, contains: []string{"--comments-file", "--commit-time", "/test all"}},
		"Prow CI contract":        {args: []string{"prow-ci", "-h"}, contains: []string{"stdin", "PULL_PULL_SHA", "3 means skip"}},
		"GHA CI contract":         {args: []string{"gha-ci", "-h"}, contains: []string{"GITHUB_OUTPUT", "CI_ENFORCE=true"}},
	} {
		t.Run(name, func(t *testing.T) {
			var output, diagnostic bytes.Buffer
			code, err := run(t.Context(), test.args, strings.NewReader(""), &output, &diagnostic)
			require.NoError(t, err)
			assert.Zero(t, code)
			assert.Empty(t, diagnostic.String())
			for _, text := range test.contains {
				assert.Contains(t, output.String(), text)
			}
			for _, text := range test.absent {
				assert.NotContains(t, output.String(), text)
			}
		})
	}
}

func TestGHAWrapperBuildFailure(t *testing.T) {
	output := filepath.Join(t.TempDir(), "output")
	// The override exercises the wrapper fallback without depending on a local build.
	text, err := os.ReadFile("gha-gate.sh")
	require.NoError(t, err)
	script := strings.Replace(string(text), `"$root/ci/select/bin/ci-select"`, `"/missing-ci-selector"`, 1)
	command := exec.CommandContext(t.Context(), "bash", "-c", script)
	command.Env = append(command.Environ(), "GITHUB_OUTPUT="+output)
	require.NoError(t, command.Run())
	text, err = os.ReadFile(output)
	require.NoError(t, err)
	assert.Equal(t, "action=run\n", string(text))
}
