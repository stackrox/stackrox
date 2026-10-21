package querybuilders

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScaleEPSSPercentToProbability(t *testing.T) {
	t.Parallel()
	// 1% and 10% are not exact in float32, which is how Scanner stores EPSS.
	// Those expected strings are the shortest decimals that round-trip that float32.
	cases := map[string]string{
		"75":     "0.75",
		">= 75":  ">=0.75",
		"> 0":    ">0",
		"<= 100": "<=1",
		"< 1":    "<0.009999999776482582",
		"0":      "0",
		"100":    "1",
		"50":     "0.5",
		"10":     "0.10000000149011612",
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, expected, scaleEPSSPercentToProbability(input))
		})
	}
}
