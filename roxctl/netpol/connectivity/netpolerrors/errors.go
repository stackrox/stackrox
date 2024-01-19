package netpolerrors

import (
	goerrors "errors"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/npg"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

var (
	// ErrYAMLMalformed is returned when YAML document cannot be parsed as such
	ErrYAMLMalformed = errors.New("YAML document is malformed")
	// ErrYAMLIsNotK8s is returned when YAML document is valid but doesn't represent a K8s/OS resource
	ErrYAMLIsNotK8s = errors.New("YAML document does not represent a K8s resource")

	errorsMapping = map[string]error{
		"the path \".*\" does not exist": errox.NotFound,
	}
	warningsMapping = map[string]error{
		"error parsing":               ErrYAMLMalformed,
		"cannot unmarshal":            ErrYAMLMalformed,
		"Object 'Kind' is missing in": ErrYAMLIsNotK8s,
	}
)

// ErrorHandler handles errors returned by k8s.io/cli-runtime/pkg/resource when reading manifests
type ErrorHandler struct {
	displayWarn           error
	treatWarningsAsErrors bool
}

// NewErrHandler returns ErrorHandler
func NewErrHandler(treatWarningsAsErrors bool) *ErrorHandler {
	return &ErrorHandler{
		displayWarn:           nil,
		treatWarningsAsErrors: treatWarningsAsErrors,
	}
}

// Warnings returns warnings to be displayed to the user
func (e *ErrorHandler) Warnings() error {
	return e.displayWarn
}

// HandleError handles errors after reading a single location with manifests
func (e *ErrorHandler) HandleError(err1 error) error {
	warn, err := e.handleErrorsWarnings(mapErrorsWarnings(disaggregate(err1)...))
	e.displayWarn = warn
	return err
}

// HandleErrorPair handles errors after reading a two locations with manifests (e.g. diff command)
func (e *ErrorHandler) HandleErrorPair(err1, err2 error) error {
	e.displayWarn = nil
	war1, e1 := mapErrorsWarnings(disaggregate(err1)...)
	war2, e2 := mapErrorsWarnings(disaggregate(err2)...)

	var err error
	e.displayWarn, err = e.handleErrorsWarnings(append(war1, war2...), append(e1, e2...))
	return err
}

func (e *ErrorHandler) handleErrorsWarnings(warnings []error, errors []error) (error, error) {
	if len(errors)+len(warnings) == 0 {
		return nil, nil
	}
	markerErr := npg.ErrErrors
	if len(errors) == 0 {
		if !e.treatWarningsAsErrors {
			return goerrors.Join(npg.ErrWarnings, goerrors.Join(warnings...)), nil
		}
		markerErr = npg.ErrWarnings
	}
	return nil, goerrors.Join(markerErr, goerrors.Join(append(errors, warnings...)...))
}

func mapErrorsWarnings(inErrs ...error) (warnings []error, outErrs []error) {
	for _, err := range inErrs {
		if err == nil {
			continue
		}
		for s, mappedWarning := range warningsMapping {
			if match, _ := regexp.Match(s, []byte(err.Error())); match {
				warnings = append(warnings, goerrors.Join(err, mappedWarning))
			}
		}
		for s, mappedError := range errorsMapping {
			if match, _ := regexp.Match(s, []byte(err.Error())); match {
				outErrs = append(outErrs, goerrors.Join(err, mappedError))
			}
		}
	}
	return warnings, outErrs
}

func disaggregate(err error) []error {
	if aggr, ok := err.(utilerrors.Aggregate); ok {
		return aggr.Errors()
	}
	return []error{err}
}
