package updater

import (
	"maps"
	"slices"
	"testing"

	"github.com/quay/claircore/libvuln/updates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterSources(t *testing.T) {
	bundles := map[string][]updates.ManagerOption{
		"alpha":   nil,
		"bravo":   nil,
		"charlie": nil,
	}

	tests := map[string]struct {
		selected  []string
		wantKeys  []string
		wantError string
	}{
		"single source": {
			selected: []string{"alpha"},
			wantKeys: []string{"alpha"},
		},
		"multiple sources": {
			selected: []string{"alpha", "charlie"},
			wantKeys: []string{"alpha", "charlie"},
		},
		"unknown source": {
			selected:  []string{"alpha", "bogus"},
			wantError: `unknown source: "bogus"`,
		},
		"all sources": {
			selected: []string{"alpha", "bravo", "charlie"},
			wantKeys: []string{"alpha", "bravo", "charlie"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := filterSources(bundles, tc.selected)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
			assert.ElementsMatch(t, tc.wantKeys, slices.Collect(maps.Keys(result)))
		})
	}
}
