package multipliers

// Normalized score into a value between 1 and maxAllowedValue.
func normalizeScore(score, saturation, maxAllowedValue float32) float32 {
	if score > saturation {
		return float32(maxAllowedValue)
	}
	return 1 + (score/saturation)*(maxAllowedValue-1)
}
