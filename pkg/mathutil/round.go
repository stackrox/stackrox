package mathutil

import "math"

// RoundToDecimal rounds a float64 number to a given precision in decimal digits.
func RoundToDecimal(n float64, d int) float64 {
	shift := math.Pow(10, float64(d))
	return math.Round(n*shift) / shift
}
