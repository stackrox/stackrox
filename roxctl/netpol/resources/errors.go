package resources

import (
	"regexp"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

var (
	warningsMapping = map[string]struct{}{
		"error parsing":               {},
		"cannot unmarshal":            {},
		"Object 'Kind' is missing in": {},
	}
)

// handleAggregatedError returns errors and warnings from the aggregated error
func handleAggregatedError(err1 error) (warn []error, err []error) {
	return recognizeWarnings(disaggregate(err1)...)
}

// recognizeWarnings reads errors returned k8s.io/cli-runtime and recognizes it as a warning or en error
func recognizeWarnings(inErrs ...error) (warnings []error, outErrs []error) {
	for _, err := range inErrs {
		if err == nil {
			continue
		}
		if isWarning(err) {
			warnings = append(warnings, err)
		} else {
			outErrs = append(outErrs, err)
		}
	}
	return warnings, outErrs
}

func isWarning(err error) bool {
	for s := range warningsMapping {
		if match, _ := regexp.Match(s, []byte(err.Error())); match {
			return true
		}
	}
	return false
}

func disaggregate(err error) []error {
	if aggr, ok := err.(utilerrors.Aggregate); ok {
		return aggr.Errors()
	}
	return []error{err}
}
