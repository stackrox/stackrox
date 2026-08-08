package policyreport

import (
	"strings"
	"unicode/utf8"
)

// maxProducerStringLength bounds any single producer-supplied string field
// (message, category, reported source, property values, ...). Long enough
// for a realistic human-readable message, short enough to bound storage and
// log volume regardless of what a misbehaving producer sends.
const maxProducerStringLength = 4096

// sanitizeProducerString trims surrounding whitespace, strips control
// characters (a misbehaving or malicious producer must not be able to inject
// them), and truncates to maxProducerStringLength. Used for every
// producer-supplied string that ends up in a SecurityEvent, per
// security-event-plan.md's "Source normalization" section ("Apply length
// limits and reject control characters").
func sanitizeProducerString(raw string) string {
	trimmed := strings.TrimSpace(raw)

	var b strings.Builder
	b.Grow(len(trimmed))
	for _, r := range trimmed {
		if r == '\t' || (r >= 0x20 && r != 0x7f) {
			b.WriteRune(r)
		}
	}
	sanitized := b.String()

	if len(sanitized) > maxProducerStringLength {
		truncated := sanitized[:maxProducerStringLength]
		for !utf8.ValidString(truncated) {
			truncated = truncated[:len(truncated)-1]
		}
		return truncated
	}
	return sanitized
}
