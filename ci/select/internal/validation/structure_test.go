package validation

import (
	"testing"

	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckStructure(t *testing.T) {
	good := rules.Mapping{
		Jobs: []string{"go", "postgres", "lint"},
		Rules: []*rules.Rule{
			{Name: "go-unit", Jobs: []string{"go"}, SkipIfOnlyChanged: []string{`^ui/`}},
			{Name: "db", Jobs: []string{"postgres"}, RunIfChanged: []string{`^central/`}},
			{Name: "lint", Jobs: []string{"lint"}, AlwaysRun: true},
		},
	}
	require.Empty(t, checkStructure(good))

	cases := map[string]struct {
		mutate func(m *rules.Mapping)
		want   string
	}{
		"empty inventory": {
			mutate: func(m *rules.Mapping) { m.Jobs = nil },
			want:   "jobs inventory must not be empty",
		},
		"duplicate rule name": {
			mutate: func(m *rules.Mapping) { m.Rules[1].Name = "go-unit" },
			want:   `duplicate rule name "go-unit"`,
		},
		"empty rule name": {
			mutate: func(m *rules.Mapping) { m.Rules[0].Name = "" },
			want:   "has an empty name",
		},
		"rule with no jobs": {
			mutate: func(m *rules.Mapping) { m.Rules[0].Jobs = nil },
			want:   "lists no jobs",
		},
		"rule with no clauses": {
			mutate: func(m *rules.Mapping) { m.Rules[0].SkipIfOnlyChanged = nil },
			want:   "declares none of run_if_changed, skip_if_only_changed, always_run",
		},
		"rule with more than one clause": {
			mutate: func(m *rules.Mapping) { m.Rules[0].RunIfChanged = []string{`^pkg/`} },
			want:   "declares more than one of run_if_changed, skip_if_only_changed, always_run",
		},
		"always_run combined with a path clause": {
			mutate: func(m *rules.Mapping) { m.Rules[2].RunIfChanged = []string{`^pkg/`} },
			want:   "declares more than one of run_if_changed, skip_if_only_changed, always_run",
		},
		"job decided twice": {
			mutate: func(m *rules.Mapping) { m.Rules[1].Jobs = []string{"go"} },
			want:   `job "go" is decided more than once`,
		},
		"unknown job in rule": {
			mutate: func(m *rules.Mapping) { m.Rules[0].Jobs = []string{"nope"} },
			want:   `names job "nope", which is not in the jobs inventory`,
		},
		"inventory job undecided": {
			mutate: func(m *rules.Mapping) { m.Jobs = append(m.Jobs, "orphan") },
			want:   `job "orphan" is in the inventory but no rule decides it`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m := good
			m.Jobs = append([]string(nil), good.Jobs...)
			m.Rules = []*rules.Rule{
				{Name: "go-unit", Jobs: []string{"go"}, SkipIfOnlyChanged: []string{`^ui/`}},
				{Name: "db", Jobs: []string{"postgres"}, RunIfChanged: []string{`^central/`}},
				{Name: "lint", Jobs: []string{"lint"}, AlwaysRun: true},
			}
			tc.mutate(&m)
			findings := checkStructure(m)
			require.NotEmpty(t, findings)
			assert.Contains(t, joinFindings(findings), tc.want)
		})
	}
}

func joinFindings(findings []string) string {
	all := ""
	for _, f := range findings {
		all += f + "\n"
	}
	return all
}
