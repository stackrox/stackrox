package csv

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatCVSSScore(t *testing.T) {
	cases := map[string]struct {
		score   *float64
		version string
		want    string
	}{
		"unavailable": {version: "UNKNOWN", want: "Not Available"},
		"available":   {score: new(7.5), version: "V3", want: "7.50 (V3)"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatCVSSScore(tc.score, tc.version))
		})
	}
}
