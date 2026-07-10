package reportcheck

import (
	"strings"
	"testing"

	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stretchr/testify/assert"
)

func packageMap(n int) map[string]*v4.Package {
	pkgs := make(map[string]*v4.Package, n)
	for i := range n {
		pkgs[string(rune('a'+i))] = &v4.Package{Name: string(rune('a' + i)), Version: "1.0.0"}
	}
	return pkgs
}

func TestIsViable(t *testing.T) {
	cases := map[string]struct {
		report      *v4.IndexReport
		wantViable  bool
		wantWarning string // substring; "" means the warning must be empty
	}{
		"should reject a nil report": {
			report:      nil,
			wantViable:  false,
			wantWarning: "nil report",
		},
		"should warn but accept a report with zero packages": {
			report: &v4.IndexReport{
				State:    "IndexFinished",
				Contents: &v4.Contents{},
			},
			wantViable:  true,
			wantWarning: "zero packages",
		},
		"should warn but accept a report with fewer packages than the minimum": {
			report: &v4.IndexReport{
				State:    "IndexFinished",
				Contents: &v4.Contents{Packages: packageMap(warnMinPackages - 1)},
			},
			wantViable:  true,
			wantWarning: "unexpectedly low",
		},
		"should accept a normal report with no warning": {
			report: &v4.IndexReport{
				State:    "IndexFinished",
				Contents: &v4.Contents{Packages: packageMap(warnMinPackages + 5)},
			},
			wantViable:  true,
			wantWarning: "",
		},
		"should warn but accept an unusually large report": {
			report: &v4.IndexReport{
				State: "IndexFinished",
				Contents: &v4.Contents{
					Packages: map[string]*v4.Package{
						// A handful of normal packages plus one with an
						// oversized field pushes the encoded report past
						// warnMaxBytes without needing thousands of entries.
						"a": {Name: "a", Version: "1.0.0"},
						"b": {Name: "b", Version: "1.0.0"},
						"c": {Name: "c", Version: "1.0.0"},
						"d": {Name: "d", Version: "1.0.0"},
						"e": {Name: "e", Version: strings.Repeat("x", warnMaxBytes+1024)},
					},
				},
			},
			wantViable:  true,
			wantWarning: "unusually large",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			viable, warning := IsViable(tc.report)
			assert.Equal(t, tc.wantViable, viable)
			if tc.wantWarning == "" {
				assert.Empty(t, warning)
			} else {
				assert.Contains(t, warning, tc.wantWarning)
			}
		})
	}
}
