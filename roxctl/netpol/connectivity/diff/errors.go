package diff

import (
	goerrors "errors"
	"regexp"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/npg"
)

var (
	errorsMapping = map[string]error{
		"the path \".*\" does not exist": errox.NotFound,
	}
	warningsMapping = map[string]error{
		"error parsing":               ErrYAMLMalformed,
		"cannot unmarshal":            ErrYAMLMalformed,
		"Object 'Kind' is missing in": ErrYAMLIsNotK8s,
	}
)

func NewErrHandler(treatWarningsAsErrors bool) *errHandler {
	return &errHandler{treatWarningsAsErrors: treatWarningsAsErrors}
}

type errHandler struct {
	treatWarningsAsErrors bool
}

func (e *errHandler) HandleErrors(err1, err2 error) error {
	e1, w1 := e.mapErrorsWarnings(err1)
	e2, w2 := e.mapErrorsWarnings(err2)
	if err := e.handleErrorsWarnings(e1, w1); err != nil {
		return err
	}
	if err := e.handleErrorsWarnings(e2, w2); err != nil {
		return err
	}
	if e1 == nil && e2 == nil && len(w1) > 0 && len(w2) > 0 && e.treatWarningsAsErrors {
		return npg.ErrErrors
	}
	return nil
}

func (e *errHandler) handleErrorsWarnings(err error, warnings []error) error {
	if err == nil && (len(warnings) == 0 || !e.treatWarningsAsErrors) {
		return nil
	}
	markerErr := npg.ErrErrors
	if e.treatWarningsAsErrors {
		markerErr = npg.ErrWarnings
	}
	return goerrors.Join(markerErr, goerrors.Join(err, goerrors.Join(warnings...)))
}

func (e *errHandler) mapErrorsWarnings(err error) (error, []error) {
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
			return goerrors.Join(err, mappedError), warnings
		}
	}
	return nil, warnings
}
