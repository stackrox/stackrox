package downloaddb

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/roxctl/common/environment"
	roxctlio "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
)

func TestDisectVersion(t *testing.T) {
	testIO, _, _, _ := roxctlio.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())
	cmd := scannerDownloadDBCommand{env: env}

	tcs := []struct {
		in   string
		want []string
		isV2 bool
	}{
		// Invalid versions
		{"", nil, false},
		{"v4.3.1", nil, false}, // leading v is 'invalid'
		{"x.3.1", nil, false},
		{"4.x.1", nil, false},
		{"4.0-1", nil, false},

		// Versions prior to Scanner V4
		{"3.99.99", nil, true},
		{"3.74.0", nil, true},
		{"4.0.0", nil, true},
		{"4.3.99", nil, true},
		{"4.3.1-1050-g8ece190c63", nil, true},
		{"4.3", nil, true},

		// Version post Scanner V4
		{"4.4", []string{"4.4"}, false},
		{"4.3.x", []string{"4.3.x", "4.3"}, false},
		{"4.3.x-1050-g8ece190c63-prerelease-ppc64le", []string{
			"4.3.x-1050-g8ece190c63-prerelease-ppc64le",
			"4.3.x-1050-g8ece190c63-prerelease",
			"4.3.x-1050-g8ece190c63",
			"4.3.x-1050",
			"4.3.x",
			"4.3",
		}, false},
		{"4.4.0", []string{"4.4.0", "4.4"}, false},
		{"4.4.99", []string{"4.4.99", "4.4"}, false},
		{"4.5.0", []string{"4.5.0", "4.5"}, false},
		{"4.5.99", []string{"4.5.99", "4.5"}, false},
		{"4.5.99-1050-blah", []string{
			"4.5.99-1050-blah",
			"4.5.99-1050",
			"4.5.99",
			"4.5",
		}, false},
	}
	for i, tc := range tcs {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			got, gotV2 := cmd.disectVersion(tc.in)
			assert.Equal(t, tc.want, got, "test: %v", tc.in)
			assert.Equal(t, tc.isV2, gotV2, fmt.Sprintf("test: %v", tc.in))
		})
	}
}
