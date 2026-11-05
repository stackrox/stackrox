package querybuilders

import (
	"strconv"
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

func TestScaleEPSSProbabilityToPercent(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"0.75":                  "75",
		">=0.75":                ">= 75",
		">= 0.75":               ">= 75",
		">0":                    "> 0",
		"<=1":                   "<= 100",
		"<0.009999999776482582": "< 1",
		"0":                     "0",
		"1":                     "100",
		"0.5":                   "50",
		"0.10000000149011612":   "10",
		"0.42":                  "42",
	}
	for input, expected := range cases {
		t.Run(input, func(t *testing.T) {
			t.Parallel()
			got, ok := ScaleEPSSProbabilityToPercent(input)
			assert.True(t, ok)
			assert.Equal(t, expected, got)
		})
	}

	rejected := map[string]string{
		"greater than one":        ">1",
		"greater or equal one":    ">=1",
		"greater than one spaced": "> 1",
		"greater or equal spaced": ">= 1",
		"above one":               "1.5",
		"negative":                "-0.1",
		"not a number":            "abc",
		"comparator only":         ">",
		"empty":                   "",
		"nan":                     "NaN",
	}
	for name, input := range rejected {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, ok := ScaleEPSSProbabilityToPercent(input)
			assert.False(t, ok)
		})
	}
}

func TestEPSSPercentRoundTrip(t *testing.T) {
	t.Parallel()
	for percent := 0; percent <= 100; percent++ {
		policy := strconv.Itoa(percent)
		got, ok := ScaleEPSSProbabilityToPercent(scaleEPSSPercentToProbability(policy))
		assert.True(t, ok, policy)
		assert.Equal(t, policy, got, policy)
	}
	for _, prefix := range []string{">=", ">", "<=", "<"} {
		for percent := 0; percent <= 100; percent++ {
			policy := prefix + " " + strconv.Itoa(percent)
			got, ok := ScaleEPSSProbabilityToPercent(scaleEPSSPercentToProbability(policy))
			if (prefix == ">" || prefix == ">=") && percent == 100 {
				assert.False(t, ok, policy)
				continue
			}
			assert.True(t, ok, policy)
			assert.Equal(t, policy, got, policy)
		}
	}
}
