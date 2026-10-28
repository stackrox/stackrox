package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	d, err := parseDefaults("# defaults\nunit run\ncheck (GOTAGS=\"\") skip\n")
	require.NoError(t, err)
	assert.Equal(t, "unit\n", d.decision(nil))
	assert.Equal(t, "skip", d.actions["check (GOTAGS=\"\")"])
	assert.Equal(t, "unit\ncheck (GOTAGS=\"\")\n", d.decision([]string{"ci-run-all-tests"}))
	assert.Equal(t, "unit\ncheck (GOTAGS=\"\")\n", d.decision([]string{`check (GOTAGS="")`}))
	assert.Equal(t, "unit\n", d.decision([]string{"unknown-job"}))
	for name, text := range map[string]string{
		"empty": "", "comments only": "# comment", "duplicate": "unit run\nunit skip",
		"inline comment": "unit run # note", "no action": "unit", "bad action": "unit maybe",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := parseDefaults(text)
			require.Error(t, err)
		})
	}
}

func TestDecision(t *testing.T) {
	jobs, err := parseDecision("unit\n# note\nunit\n")
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"unit": true}, jobs)
	jobs, err = parseDecision(formatDecision(nil))
	require.NoError(t, err)
	assert.Empty(t, jobs)
	for name, text := range map[string]string{"blank": " \n", "inline comment": "unit # note"} {
		t.Run(name, func(t *testing.T) {
			_, err := parseDecision(text)
			require.Error(t, err)
		})
	}
}

func TestGateAction(t *testing.T) {
	d := defaults{actions: map[string]string{"unit": "run", "e2e": "skip"}}
	cases := map[string]struct {
		job             string
		jobs            map[string]bool
		enforce, forced bool
		want            string
	}{
		"shadow runs":      {job: "unit", jobs: map[string]bool{}, want: "run"},
		"listed":           {job: "unit", jobs: map[string]bool{"unit": true}, enforce: true, want: "run"},
		"absent":           {job: "unit", jobs: map[string]bool{}, enforce: true, want: "skip"},
		"outside":          {job: "unknown", jobs: map[string]bool{}, enforce: true, want: "run"},
		"missing decision": {job: "unit", enforce: true, want: "run"},
		"skip default":     {job: "e2e", enforce: true, want: "skip"},
		"test override":    {job: "e2e", jobs: map[string]bool{}, enforce: true, forced: true, want: "run"},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, gateAction(test.job, test.jobs, d, test.enforce, test.forced))
		})
	}
}
