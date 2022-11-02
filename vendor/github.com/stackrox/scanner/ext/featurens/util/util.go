package util

// NormalizeOSName normalizes the given OS name for consistency.
func NormalizeOSName(os string) string {
	switch os {
	case "ol", "oracle":
		return "oracle"
	default:
		return os
	}
}
