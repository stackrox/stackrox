package common

import (
	"strings"

	"github.com/quay/claircore"
)

// NormalizeSeverity maps Red Hat severity strings to claircore's normalized
// serverity levels.
func NormalizeSeverity(severity string) claircore.Severity {
	switch strings.ToLower(severity) {
	case "none":
		return claircore.Unknown
	case "low":
		return claircore.Low
	case "moderate":
		return claircore.Medium
	case "important":
		return claircore.High
	case "critical":
		return claircore.Critical
	default:
		return claircore.Unknown
	}
}
