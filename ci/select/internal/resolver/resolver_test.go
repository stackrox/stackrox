package resolver

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stretchr/testify/require"
)

// mapping builds a compiled mapping for the decision tests. It fails the
// test if any pattern does not compile, since the resolver trusts validated
// input.
func mapping(t *testing.T, m rules.Mapping) rules.Mapping {
	t.Helper()
	for _, r := range m.Rules {
		require.NoError(t, r.Compile())
	}
	return m
}

func TestResolve(t *testing.T) {
	m := func(t *testing.T) rules.Mapping {
		return mapping(t, rules.Mapping{
			Version:   1,
			Jobs:      []string{"go", "postgres", "sensor", "ui-e2e", "always"},
			AlwaysRun: []string{"always"},
			Rules: []*rules.Rule{
				{Name: "go-unit", Jobs: []string{"go"}, SkipIfOnlyChanged: []string{`^ui/`, `^.*\.md$`}},
				{Name: "db", Jobs: []string{"postgres"}, RunIfChanged: []string{`^central/`, `^pkg/`}},
				{Name: "both", Jobs: []string{"sensor"}, RunIfChanged: []string{`^sensor/critical/`}, SkipIfOnlyChanged: []string{`^sensor/`}},
				{Name: "ui-e2e", Jobs: []string{"ui-e2e"}, SkipIfOnlyChanged: []string{`^.*\.md$`}},
			},
		})
	}

	cases := map[string]struct {
		files []string
		want  map[string]string
	}{
		"empty diff runs everything": {
			files: nil,
			want:  map[string]string{"go": "run", "postgres": "run", "sensor": "run", "ui-e2e": "run", "always": "run"},
		},
		"docs only": {
			files: []string{"README.md"},
			want:  map[string]string{"go": "skip", "postgres": "skip", "sensor": "run", "ui-e2e": "skip", "always": "run"},
		},
		"ui only": {
			files: []string{"ui/App.tsx"},
			want:  map[string]string{"go": "skip", "postgres": "skip", "sensor": "run", "ui-e2e": "run", "always": "run"},
		},
		"central change": {
			files: []string{"central/sac/client.go"},
			want:  map[string]string{"go": "run", "postgres": "run", "sensor": "run", "ui-e2e": "run", "always": "run"},
		},
		"run_if_changed with no match skips": {
			files: []string{"scanner/x.go"},
			want:  map[string]string{"go": "run", "postgres": "skip", "sensor": "run", "ui-e2e": "run", "always": "run"},
		},
		"both clauses: skip set covers diff, no run match -> skip": {
			files: []string{"sensor/kubernetes/fake.go"},
			want:  map[string]string{"go": "run", "postgres": "skip", "sensor": "skip", "ui-e2e": "run", "always": "run"},
		},
		"both clauses: run match wins over skip coverage": {
			files: []string{"sensor/critical/reconcile.go"},
			want:  map[string]string{"go": "run", "postgres": "skip", "sensor": "run", "ui-e2e": "run", "always": "run"},
		},
		"both clauses: file outside skip set -> run": {
			files: []string{"sensor/kubernetes/fake.go", "pkg/x.go"},
			want:  map[string]string{"go": "run", "postgres": "run", "sensor": "run", "ui-e2e": "run", "always": "run"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			d := Resolve(tc.files, m(t))
			require.Equal(t, tc.want, d.Actions)
		})
	}
}

func TestExclusionPattern(t *testing.T) {
	m := mapping(t, rules.Mapping{
		Version: 1,
		Jobs:    []string{"go"},
		Rules: []*rules.Rule{
			// Skip go when only docs change, but a prefetch list is not a doc.
			{Name: "go", Jobs: []string{"go"}, SkipIfOnlyChanged: []string{
				`^.*\.txt$`, `!^tests/images-to-prefetch\.txt$`,
			}},
		},
	})

	require.Equal(t, "skip", Resolve([]string{"notes.txt"}, m).Action("go"))
	require.Equal(t, "run", Resolve([]string{"tests/images-to-prefetch.txt"}, m).Action("go"))
	require.Equal(t, "run", Resolve([]string{"notes.txt", "tests/images-to-prefetch.txt"}, m).Action("go"))
}

func TestFallbackRunsEverything(t *testing.T) {
	d := Fallback(errors.New("boom"))
	require.Nil(t, d.Actions)
	require.Equal(t, "run", d.Action("anything"))
	require.Contains(t, d.Log, "selection unavailable")
}
