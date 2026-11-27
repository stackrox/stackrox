package vuln

import (
	"testing"

	"github.com/quay/claircore"
	"github.com/quay/claircore/toolkit/types"
	"github.com/stretchr/testify/assert"
)

func TestIgnoreVulnerability(t *testing.T) {
	t.Setenv("ROX_SCANNER_V4_SKIP_UNUSED_NOT_AFFECTED", "true")

	cases := map[string]struct {
		vuln *claircore.Vulnerability
		want bool
	}{
		"nil": {
			vuln: nil,
			want: false,
		},
		"affected binary package is kept": {
			vuln: &claircore.Vulnerability{Package: &claircore.Package{Kind: types.BinaryPackage}},
			want: false,
		},
		"not-affected ancestry package is kept": {
			vuln: &claircore.Vulnerability{Invert: true, Package: &claircore.Package{Kind: types.AncestryPackage}},
			want: false,
		},
		"not-affected binary (RPM) package is dropped": {
			vuln: &claircore.Vulnerability{Invert: true, Package: &claircore.Package{Kind: types.BinaryPackage}},
			want: true,
		},
		"not-affected with no package is dropped": {
			vuln: &claircore.Vulnerability{Invert: true},
			want: true,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, ignoreVulnerability(tc.vuln))
		})
	}
}

func TestIgnoreVulnerability_Disabled(t *testing.T) {
	t.Setenv("ROX_SCANNER_V4_SKIP_UNUSED_NOT_AFFECTED", "false")
	// Even a not-affected RPM record is kept when the filter is disabled.
	v := &claircore.Vulnerability{Invert: true, Package: &claircore.Package{Kind: types.BinaryPackage}}
	assert.False(t, ignoreVulnerability(v))
}
