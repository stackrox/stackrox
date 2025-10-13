package manual

import (
	"strings"

	"github.com/quay/claircore"
)

func severity(severity string) claircore.Severity {
	switch {
	case strings.EqualFold(severity, "Low"):
		return claircore.Low
	case strings.EqualFold(severity, "Medium"):
		return claircore.Medium
	case strings.EqualFold(severity, "Important"):
		return claircore.High
	case strings.EqualFold(severity, "Critical"):
		return claircore.Critical
	default:
		return claircore.Unknown
	}
}
