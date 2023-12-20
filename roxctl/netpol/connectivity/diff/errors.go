package diff

import (
	goerrors "errors"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/npg"
)

var (
	// errYAMLMalformed is returned when YAML document cannot be parsed as such
	errYAMLMalformed = errors.New("YAML document is malformed")
	// errYAMLIsNotK8s is returned when YAML document is valid but doesn't represent a K8s/OS resource
	errYAMLIsNotK8s = errors.New("YAML document does not represent a K8s resource")

	errorsMapping = map[string]error{
		"the path \".*\" does not exist": errox.NotFound,
	}
	warningsMapping = map[string]error{
		"error parsing":               errYAMLMalformed,
		"cannot unmarshal":            errYAMLMalformed,
		"Object 'Kind' is missing in": errYAMLIsNotK8s,
	}
)

func newErrHandler(treatWarningsAsErrors bool) *errHandler {
	return &errHandler{treatWarningsAsErrors: treatWarningsAsErrors}
}

type errHandler struct {
	treatWarningsAsErrors bool
}

func (e *errHandler) HandleErrors(err1, err2 error) error {
	w1, e1 := e.mapErrorsWarnings(err1)
	w2, e2 := e.mapErrorsWarnings(err2)
	if err := e.handleErrorsWarnings(w1, e1); err != nil {
		return err
	}
	if err := e.handleErrorsWarnings(w2, e2); err != nil {
		return err
	}
	if e1 == nil && e2 == nil && len(w1)+len(w2) > 0 && e.treatWarningsAsErrors {
		return npg.ErrErrors
	}
	return nil
}

func (e *errHandler) handleErrorsWarnings(warnings []error, err error) error {
	if err == nil && (len(warnings) == 0 || !e.treatWarningsAsErrors) {
		return nil
	}
	markerErr := npg.ErrErrors
	if e.treatWarningsAsErrors {
		markerErr = npg.ErrWarnings
	}
	return goerrors.Join(markerErr, goerrors.Join(err, goerrors.Join(warnings...)))
}

func (e *errHandler) mapErrorsWarnings(err error) ([]error, error) {
	if err == nil {
		return nil, nil
	}
	warnings := make([]error, 0)
	for s, mappedWarning := range warningsMapping {
		if match, _ := regexp.Match(s, []byte(err.Error())); match {
			warnings = append(warnings, goerrors.Join(err, mappedWarning))
		}
	}

	for s, mappedError := range errorsMapping {
		if match, _ := regexp.Match(s, []byte(err.Error())); match {
			return warnings, goerrors.Join(err, mappedError)
		}
	}
	return warnings, nil
}
