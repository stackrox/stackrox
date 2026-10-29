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

func e2eTargets() []string {
	return []string{
		"e2e-byodb-tests", "e2e-db-backup-restore-tests-gke", "e2e-db-backup-restore-tests-gke-konflux",
		"e2e-gke-upgrade-tests", "e2e-nongroovy-tests-gke", "e2e-nongroovy-tests-gke-konflux",
		"e2e-qa-tests-gke", "e2e-qa-tests-gke-konflux", "e2e-qa-tests-ocp-fips-konflux",
		"e2e-ui-tests-gke", "e2e-ui-tests-gke-konflux",
	}
}

func TestPolicy(t *testing.T) {
	t.Parallel()
	m, err := loadMapping("../test-selection-rules.yaml")
	require.NoError(t, err)
	require.ElementsMatch(t, append([]string{"go", "go-postgres", "go-bench", "sensor-integration-tests"}, e2eTargets()...), m.Jobs)
	cases := map[string]struct {
		files []string
		skip  []string
	}{
		"should skip backend targets for UI only": {
			files: []string{"ui/example.tsx"}, skip: []string{"go", "go-postgres", "go-bench", "sensor-integration-tests"},
		},
		"should skip only backend targets for UI and root docs": {
			files: []string{"ui/example.tsx", "README.md"}, skip: []string{"go", "go-postgres", "go-bench", "sensor-integration-tests"},
		},
		"should skip all for docs": {files: []string{"docs/guide.md"}, skip: append([]string{"go", "go-postgres", "go-bench", "sensor-integration-tests"}, e2eTargets()...)},
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
		"should skip all for nested Markdown docs": {
			files: []string{"central/example/guide.md"},
			skip:  append([]string{"go", "go-postgres", "go-bench", "sensor-integration-tests"}, e2eTargets()...),
		},
		"should run all for empty diff":                          {},
		"should not match backend roots within other code paths": {files: []string{"other/docs/example.go"}},
		"should keep backend tests after a Markdown fixture rename": {
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

func TestDocumentationPolicy(t *testing.T) {
	t.Parallel()
	m, err := loadMapping("../test-selection-rules.yaml")
	require.NoError(t, err)
	backend := []string{"go", "go-postgres", "go-bench", "sensor-integration-tests"}
	all := append(slices.Clone(backend), e2eTargets()...)
	for name, c := range map[string]struct {
		files []string
		skip  []string
	}{
		"root Markdown and text":                    {files: []string{"README.md", "NOTICE.txt"}, skip: all},
		"nested and uppercase extensions":           {files: []string{"pkg/README.MD", "sensor/notes.TxT"}, skip: all},
		"newline filename":                          {files: []string{"docs/multi\nline.md"}, skip: all},
		"mixed documentation and Go":                {files: []string{"README.md", "pkg/example.go"}},
		"UI implementation":                         {files: []string{"README.md", "ui/src/App.tsx"}, skip: backend},
		"documentation without supported extension": {files: []string{"docs/guide.html"}},
		"extension followed by suffix":              {files: []string{"README.md.bak"}},
		"empty diff":                                {},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := resolve(c.files, m)
			for _, target := range all {
				want := "run"
				if slices.Contains(c.skip, target) {
					want = "skip"
				}
				assert.Equal(t, want, d.action(target), target)
			}
		})
	}
}

func TestDocumentationInputs(t *testing.T) {
	t.Parallel()
	m, err := loadMapping("../test-selection-rules.yaml")
	require.NoError(t, err)
	for path, skip := range map[string][]string{
		"docs/testdata/input.md":                                                                 nil,
		"tests/images-to-prefetch.txt":                                                           nil,
		"qa-tests-backend/scripts/images-to-prefetch.txt":                                        nil,
		"sensor/tests/images-to-prefetch.txt":                                                    {"go-postgres", "go-bench"},
		"roxctl/common/report/testdata/passed.txt":                                               nil,
		"central/tlsconfig/testdata/cert.txt":                                                    nil,
		"tests/e2e/bats/fixtures/panic-log-check-output.txt":                                     nil,
		"tests/roxctl/bats-tests/local/expect/roxctl--help.txt":                                  nil,
		"qa-tests-backend/test-images/lineage/dummy.txt":                                         nil,
		"sensor/common/centralclient/testdata/central/info.md":                                   nil,
		"pkg\nname/testdata/fixture.md":                                                          nil,
		"roxctl/netpol/connectivity/diff/testdata/acs-security-demos-new-version/diff_output.md": nil,
		"tools/boilerplate.go.txt":                                                               nil,
		"operator/hack/boilerplate.go.txt":                                                       {"go-postgres", "go-bench", "sensor-integration-tests"},
		"config-controller/hack/boilerplate.go.txt":                                              nil,
		".openshift-ci/dev-requirements.txt":                                                     nil,
		"tests/performance/scripts/opensearch-upload/requirements.txt":                           nil,
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			for _, files := range [][]string{{path}, {"README.md", path}} {
				d := resolve(files, m)
				for _, target := range m.Jobs {
					want := "run"
					if slices.Contains(skip, target) {
						want = "skip"
					}
					assert.Equal(t, want, d.action(target), target)
				}
			}
		})
	}
}

func TestRootedPathAlternatives(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		paths, path, want string
	}{
		"should match the first alternative at the root": {
			paths: `['^ui/|docs/']`, path: "ui/example.tsx", want: "skip",
		},
		"should match a later alternative at the root": {
			paths: `['^ui/|docs/']`, path: "docs/guide.md", want: "skip",
		},
		"should not match a later alternative inside a path": {
			paths: `['^ui/|docs/']`, path: "pkg/docs/input.go", want: "run",
		},
		"should not treat a nested Markdown fixture as rooted documentation": {
			paths: `['^ui/|\.md$']`, path: "pkg/testdata/output.md", want: "run",
		},
		"should not let multiline flags escape the root anchor": {
			paths: `['^ui/|(?m)^docs/']`, path: "pkg/\ndocs/input.go", want: "run",
		},
		"should apply a later exclusion alternative at the root": {
			paths: `['^docs/', '!^docs/private/|docs/generated/']`, path: "docs/generated/guide.md", want: "run",
		},
		"should not apply a later exclusion alternative inside a path": {
			paths: `['^docs/', '!^docs/private/|docs/generated/']`, path: "docs/example/docs/generated/guide.md", want: "skip",
		},
		"should preserve quoted literal patterns": {
			paths: `['^docs/\Qliteral']`, path: "docs/literal", want: "skip",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rules := strings.Replace(validRules, "['^docs/']", c.paths, 1)
			m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), rules))
			require.NoError(t, err)
			assert.Equal(t, c.want, resolve([]string{c.path}, m).action("go"))
		})
	}
}

func TestPathExclusions(t *testing.T) {
	t.Parallel()
	for name, paths := range map[string]string{
		"exclusions after includes":  "['^docs/', '^docs/testdata/', '!^docs/testdata/']",
		"exclusions before includes": "['!^docs/testdata/', '^docs/testdata/', '^docs/']",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data := strings.Replace(validRules, "['^docs/']", paths, 1)
			m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules"), data))
			require.NoError(t, err)
			assert.Equal(t, "go skip\npostgres skip\n", resolve([]string{"docs/guide.md"}, m).String())
			assert.Equal(t, "go run\npostgres run\n", resolve([]string{"docs/testdata/input.txt"}, m).String())
			assert.Equal(t, "go run\npostgres run\n", resolve([]string{"docs/guide.md", "docs/testdata/input.txt"}, m).String())
			assert.Equal(t, "go run\npostgres run\n", resolve(nil, m).String())

			data = strings.Replace(data, "  - name: remaining", `  - name: test-inputs
    when: safe-paths
    paths: ['^docs/testdata/']
    skip: [go]
  - name: remaining`, 1)
			m, err = loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules"), data))
			require.NoError(t, err)
			forward := resolve([]string{"docs/guide.md", "docs/testdata/input.txt"}, m)
			assert.Equal(t, "go skip\npostgres run\n", forward.String(), "exclusions apply only to their own rule")
			slices.Reverse(m.Rules[:len(m.Rules)-1])
			assert.Equal(t, forward, resolve([]string{"docs/guide.md", "docs/testdata/input.txt"}, m))
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
	data, err := os.ReadFile("../test-selection-rules.yaml")
	require.NoError(t, err)
	// Enrollment alone grants no path-based skip permissions.
	data = []byte(strings.Replace(string(data), "jobs:\n", "jobs:\n  - ui\n", 1))
	m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules"), string(data)))
	require.NoError(t, err)
	for name, files := range map[string][]string{
		"frontend":         {"ui/src/App.tsx"},
		"docs":             {"docs/guide.md", "README.md"},
		"nested text":      {"central/notes.TXT", "sensor/README.MD"},
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
