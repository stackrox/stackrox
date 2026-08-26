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

// testWarnMaxBytes stands in for the caller-supplied threshold that, in
// production, is derived from env.VirtualMachinesPullMaxResponseSizeKB.
const testWarnMaxBytes = 2 << 20 // 2 MiB

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
		"should accept a normal report with no warning": {
			report: &v4.IndexReport{
				State:    "IndexFinished",
				Contents: &v4.Contents{Packages: packageMap(5)},
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
						// testWarnMaxBytes without needing thousands of entries.
						"a": {Name: "a", Version: "1.0.0"},
						"b": {Name: "b", Version: "1.0.0"},
						"c": {Name: "c", Version: "1.0.0"},
						"d": {Name: "d", Version: "1.0.0"},
						"e": {Name: "e", Version: strings.Repeat("x", testWarnMaxBytes+1024)},
					},
				},
			},
			wantViable:  true,
			wantWarning: "report is",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			viable, warning := IsViable(tc.report, testWarnMaxBytes)
			assert.Equal(t, tc.wantViable, viable)
			if tc.wantWarning == "" {
				assert.Empty(t, warning)
			} else {
				assert.Contains(t, warning, tc.wantWarning)
			}
		})
	}
}
