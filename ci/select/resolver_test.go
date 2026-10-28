package main

import (
	"path/filepath"
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
	require.Len(t, m.Rules, 6)
	cases := map[string]struct {
		files []string
		skip  []string
	}{
		"should skip all for UI and root docs": {
			files: []string{"ui/example.tsx", "README.md"}, skip: m.Jobs,
		},
		"should skip all for docs": {files: []string{"docs/guide.md"}, skip: m.Jobs},
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
		"should run all for Central and a root README": {files: []string{"central/sac/example.go", "README.md"}},
		"should not exempt backend Markdown fixtures":  {files: []string{"central/example/fixture.md"}},
		"should run all for empty diff":                {},
		"should not match roots within other paths":    {files: []string{"other/docs/guide.md"}},
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
    when: every-file-matches
    paths: ['^docs/', '^ui/']
    skip: [postgres]
  - name: remaining`, 1)
	data = strings.Replace(data, "skip: ['*']", "skip: [go]", 1)
	m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules"), data))
	require.NoError(t, err)
	d := resolve([]string{"docs/a.md", "docs/b.md"}, m)
	assert.Equal(t, "go skip\npostgres skip\n", d.String())
	assert.Contains(t, d.log, `go: skip by "docs"`)
	assert.Contains(t, d.log, `postgres: skip by "second"`)
	d = resolve([]string{"docs/a.md", "ui/app.tsx"}, m)
	assert.Equal(t, "go run\npostgres skip\n", d.String())
	assert.Contains(t, d.log, `go: run by "remaining"`)
	d = resolve(nil, m)
	assert.Equal(t, "go run\npostgres run\n", d.String())
	assert.Contains(t, d.log, "empty diff")
}
