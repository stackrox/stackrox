package partial

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/quay/claircore"
	"github.com/stretchr/testify/assert"
)

func TestEnrich(t *testing.T) {
	tcs := []struct{
		name     string
		vr       *claircore.VulnerabilityReport
		expected []string
	}{
		{
			name: "basic",
			vr: &claircore.VulnerabilityReport{
				Packages: map[string]*claircore.Package{
					"0": {
						PackageDB: "nodejs:app/js",
					},
					"1": {
						PackageDB: "go:app/go",
					},
					"2": {
						PackageDB: "nodejs:app2/js",
					},
					"3": {
						PackageDB: "nodejs:app3/js",
					},
					"4": {
						PackageDB: "go:app2/go",
					},
					"5": {
						PackageDB: "nodejs:app4/js",
					},
				},
				PackageVulnerabilities: map[string][]string{
					"0": {"0"},
					"1": {"5"},
					"2": {"0", "1", "2", "3", "4"},
					"5": {},
				},
			},
			expected: []string{"3", "5"},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var e Enricher
			_, m, err := e.Enrich(context.Background(), nil, tc.vr)
			assert.NoError(t, err)

			var got []string
			err = json.Unmarshal(m[0], &got)
			assert.NoError(t, err)

			assert.Equal(t, tc.expected, got)
		})
	}
}
