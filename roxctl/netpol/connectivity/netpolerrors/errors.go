package netpolerrors

import (
	goerrors "errors"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/npg"
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

func NewErrHandler(treatWarningsAsErrors bool, logger logger.Logger) *ErrHandler {
	return &ErrHandler{treatWarningsAsErrors: treatWarningsAsErrors, logger: logger}
}

type ErrHandler struct {
	treatWarningsAsErrors bool
	logger                logger.Logger
}

func (e *ErrHandler) HandleError(err error) error {
	return e.handleErrorsWarnings(e.mapErrorsWarnings(err))
}

func (e *ErrHandler) HandleErrorPair(err1, err2 error) error {
	w1, e1 := e.mapErrorsWarnings(err1)
	w2, e2 := e.mapErrorsWarnings(err2)
	return e.handleErrorsWarnings(append(w1, w2...), goerrors.Join(e1, e2))
}

func (e *ErrHandler) handleErrorsWarnings(warnings []error, err error) error {
	if err == nil && (len(warnings) == 0) {
		return nil
	}
	if err == nil && !e.treatWarningsAsErrors {
		resErr := goerrors.Join(npg.ErrWarnings, goerrors.Join(warnings...))
		e.logger.WarnfLn("%s", resErr.Error())
		return nil
	}
	markerErr := npg.ErrErrors
	if err == nil {
		markerErr = npg.ErrWarnings
	}
	return goerrors.Join(markerErr, goerrors.Join(err, goerrors.Join(warnings...)))
}

func (e *ErrHandler) mapErrorsWarnings(err error) ([]error, error) {
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
