package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicy(t *testing.T) {
	t.Parallel()
	m, err := loadMapping("../test-domains.yaml")
	require.NoError(t, err)
	require.Equal(t, []string{"go", "go-postgres", "go-bench", "sensor-integration-tests"}, m.Jobs)
	cases := map[string]struct {
		files []string
		skip  []string
	}{
		"should skip all for UI and root docs": {
			files: []string{"ui/example.tsx", "README.md"}, skip: []string{"go", "go-postgres", "go-bench", "sensor-integration-tests"},
		},
		"should skip all for docs": {files: []string{"docs/guide.md"}, skip: []string{"go", "go-postgres", "go-bench", "sensor-integration-tests"}},
		"should keep database jobs for Central": {
			files: []string{"central/sac/example.go"}, skip: []string{"sensor-integration-tests"},
		},
		"should keep Sensor tests for Sensor": {
			files: []string{"sensor/tests/example_test.go"}, skip: []string{"go-postgres", "go-bench"},
		},
		"should keep Go for Scanner": {
			files: []string{"scanner/example.go"}, skip: []string{"go-postgres", "go-bench", "sensor-integration-tests"},
		},
		"should keep Go for Operator implementation": {
			files: []string{"operator/internal/reconcile/example.go"},
			skip:  []string{"go-postgres", "go-bench", "sensor-integration-tests"},
		},
		"should keep Go for Operator root files": {
			files: []string{"operator/example.go"},
			skip:  []string{"go-postgres", "go-bench", "sensor-integration-tests"},
		},
		"should run shared API consumers for mixed Operator changes": {
			files: []string{"operator/internal/example.go", "operator/api/v1alpha1/central_types.go"},
		},
		"should keep shared API consumers":            {files: []string{"operator/api/v1alpha1/central_types.go"}},
		"should require review of new Operator roots": {files: []string{"operator/new-domain/input.go"}},
		"should run all for Central and Sensor": {
			files: []string{"central/sac/example.go", "sensor/tests/example_test.go"},
		},
		"should compose Central and root README coverage": {
			files: []string{"central/sac/example.go", "README.md"}, skip: []string{"sensor-integration-tests"},
		},
		"should compose Scanner and Operator coverage": {
			files: []string{"scanner/example.go", "operator/internal/example.go"},
			skip:  []string{"go-postgres", "go-bench", "sensor-integration-tests"},
		},
		"should keep targets with an uncovered shared path": {
			files: []string{"scanner/example.go", "operator/internal/example.go", "pkg/shared.go"},
		},
		"should not exempt backend Markdown fixtures": {files: []string{"central/example/fixture.md"}},
		"should run all for empty diff":               {},
		"should not match roots within other paths":   {files: []string{"other/docs/guide.md"}},
		"should not treat a renamed backend fixture as docs only": {
			files: []string{"pkg/testdata/input.md", "docs/input.md"},
		},
	}
	for _, path := range []string{"pkg/features/example.go", "go.mod", "proto/example.proto", "generated/api/example.go",
		"Makefile", ".github/workflows/unit-tests.yaml", "sensor/common/example.go", "central/sensor/example.go"} {
		cases["should run all for "+path] = struct {
			files []string
			skip  []string
		}{files: []string{path}}
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := resolve(c.files, m)
			for _, job := range m.Jobs {
				expected := "run"
				for _, skipped := range c.skip {
					if skipped == job {
						expected = "skip"
					}
				}
				assert.Equal(t, expected, d.action(job), job)
			}
			assert.Equal(t, "run", d.action("unenrolled-job"))
		})
	}
}

func TestResolve(t *testing.T) {
	t.Parallel()
	data := strings.Replace(validRules, "  - name: remaining", `  - name: second
    when: safe-paths
    paths: ['^docs/', '^ui/']
    skip: [postgres]
  - name: remaining`, 1)
	data = strings.Replace(data, "skip: [go, postgres]", "skip: [go]", 1)
	m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules"), data))
	require.NoError(t, err)
	d := resolve([]string{"docs/a.md", "docs/b.md"}, m)
	assert.Equal(t, "go skip\npostgres skip\n", d.String())
	assert.Contains(t, d.log, `go: skip by ["docs"]`)
	assert.Contains(t, d.log, `postgres: skip by ["second"]`)
	d = resolve([]string{"docs/a.md", "ui/app.tsx"}, m)
	assert.Equal(t, "go run\npostgres skip\n", d.String())
	assert.Contains(t, d.log, `go: run by "remaining"`)
	d = resolve(nil, m)
	assert.Equal(t, "go run\npostgres run\n", d.String())
	assert.Contains(t, d.log, "empty diff")
}

func TestNewTarget(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../test-domains.yaml")
	require.NoError(t, err)
	// Enrollment alone grants no path-based skip permissions.
	data = []byte(strings.Replace(string(data), "jobs: [", "jobs: [ui, ", 1))
	m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules"), string(data)))
	require.NoError(t, err)
	for name, files := range map[string][]string{
		"frontend":         {"ui/src/App.tsx"},
		"docs":             {"docs/guide.md", "README.md"},
		"combined domains": {"scanner/example.go", "operator/internal/example.go"},
		"empty":            nil,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := resolve(files, m)
			assert.Equal(t, "run", d.action("ui"))
			assert.Contains(t, d.log, `ui: run by "remaining"`)
		})
	}
}

func TestCoverageOrder(t *testing.T) {
	t.Parallel()
	data := strings.Replace(validRules, "  - name: remaining", `  - name: frontend
    when: safe-paths
    paths: ['^ui/', '^docs/']
    skip: [postgres]
  - name: remaining`, 1)
	m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules"), data))
	require.NoError(t, err)
	for name, c := range map[string]struct {
		files []string
		want  string
	}{
		"overlap":        {[]string{"docs/a.md", "docs/b.md"}, "go skip\npostgres skip\n"},
		"joint coverage": {[]string{"docs/a.md", "ui/app.tsx"}, "go run\npostgres skip\n"},
		"uncovered":      {[]string{"docs/a.md", "ui/app.tsx", "pkg/shared.go"}, "go run\npostgres run\n"},
		"empty":          {nil, "go run\npostgres run\n"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			forward := resolve(c.files, m)
			reversed := m
			reversed.Rules = slices.Clone(m.Rules)
			slices.Reverse(reversed.Rules[:len(reversed.Rules)-1])
			backward := resolve(c.files, reversed)
			assert.Equal(t, c.want, forward.String())
			assert.Equal(t, forward, backward)
			if name == "joint coverage" {
				assert.Contains(t, forward.log, `postgres: skip by ["docs" "frontend"]`)
			}
		})
	}
}
