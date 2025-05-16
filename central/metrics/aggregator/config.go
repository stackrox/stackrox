package aggregator

import (
	"errors"
	"regexp"
)

var metricNamePattern = regexp.MustCompile("^[a-zA-Z0-9_]+$")

func validateMetricName(s string) error {
	if len(s) == 0 {
		return errors.New("empty")
	}
	if !metricNamePattern.MatchString(s) {
		return errors.New("bad characters")
	}
	return nil
}
