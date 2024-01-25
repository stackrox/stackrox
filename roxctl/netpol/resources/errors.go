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

// ErrorHandler handles errors returned by k8s.io/cli-runtime/pkg/resource when reading manifests
type ErrorHandler struct {
	treatWarningsAsErrors bool
}

// NewErrHandler returns ErrorHandler
func NewErrHandler(treatWarningsAsErrors bool) *ErrorHandler {
	return &ErrorHandler{
		treatWarningsAsErrors: treatWarningsAsErrors,
	}
}

// HandleError returns errors and warnings from the aggregated error
func (e *ErrorHandler) HandleError(err1 error) (warn []error, err []error) {
	flatErrs := disaggregate(err1)
	w, er := recognizeWarnings(flatErrs...)
	return e.classifyErrors(w, er)
}

// classifyErrors assigns an error to either warnings or errors depending on user-provided settings
func (e *ErrorHandler) classifyErrors(inWarnings []error, inErrors []error) (outWarn []error, outErr []error) {
	if len(inErrors)+len(inWarnings) == 0 {
		return
	}
	outErr = inErrors
	outWarn = inWarnings

	if e.treatWarningsAsErrors {
		return []error{}, append(outErr, inWarnings...)
	}
	return outWarn, outErr
}

// recognizeWarnings reads errors returned k8s.io/cli-runtime and decides about treating it as a warning or error
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
