package stringutils

// OrDefault returns the string if it's not empty, or the default.
func OrDefault(s, defaultString string) string {
	if s != "" {
		return s
	}
	return defaultString
}

// PointerOrDefault returns the string if it's not nil nor empty, or the default.
func PointerOrDefault(s *string, defaultString string) string {
	if s == nil {
		return defaultString
	}

	return OrDefault(*s, defaultString)
}
