package updater

import (
	"testing"

	"github.com/quay/claircore/libvuln/updates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterSources(t *testing.T) {
	allSources := map[string][]updates.ManagerOption{
		"alpine":             nil,
		"aws":                nil,
		"debian":             nil,
		"epss":               nil,
		"manual":             nil,
		"nvd":                nil,
		"oracle":             nil,
		"osv":                nil,
		"photon":             nil,
		"rhel-vex":           nil,
		"stackrox-rhel-csaf": nil,
		"suse":               nil,
		"ubuntu":             nil,
	}

	tests := map[string]struct {
		filter    string
		wantKeys  []string
		wantError string
	}{
		"single source": {
			filter:   "alpine",
			wantKeys: []string{"alpine"},
		},
		"multiple sources": {
			filter:   "alpine,nvd,osv",
			wantKeys: []string{"alpine", "nvd", "osv"},
		},
		"whitespace trimming": {
			filter:   " alpine , nvd ",
			wantKeys: []string{"alpine", "nvd"},
		},
		"unknown source": {
			filter:    "alpine,bogus",
			wantError: `unknown source: "bogus"`,
		},
		"all sources": {
			filter:   "alpine,aws,debian,epss,manual,nvd,oracle,osv,photon,rhel-vex,stackrox-rhel-csaf,suse,ubuntu",
			wantKeys: []string{"alpine", "aws", "debian", "epss", "manual", "nvd", "oracle", "osv", "photon", "rhel-vex", "stackrox-rhel-csaf", "suse", "ubuntu"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := filterSources(allSources, tc.filter)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			var gotKeys []string
			for k := range result {
				gotKeys = append(gotKeys, k)
			}
			assert.ElementsMatch(t, tc.wantKeys, gotKeys)
		})
	}
}
