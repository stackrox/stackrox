package resources

import (
	npganalyzer "github.com/np-guard/cluster-topology-analyzer/v2/pkg/analyzer"
	npgconnlist "github.com/np-guard/netpol-analyzer/pkg/netpol/connlist"
	npgdiff "github.com/np-guard/netpol-analyzer/pkg/netpol/diff"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/npg"
)

type standardNPGuardError struct {
	err      error
	location string
	isSevere bool
}

func (e *standardNPGuardError) Error() error {
	return e.err
}

func (e *standardNPGuardError) Location() string {
	return e.location
}

func (e *standardNPGuardError) IsSevere() bool {
	return e.isSevere
}

func ConvertDiffError(input []npgdiff.DiffError) []*standardNPGuardError {
	errArr := make([]*standardNPGuardError, len(input))
	for i, err := range input {
		errArr[i] = &standardNPGuardError{
			err:      err.Error(),
			location: err.Location(),
			isSevere: err.IsSevere(),
		}
	}
	return errArr
}

func ConvertConnlistError(input []npgconnlist.ConnlistError) []*standardNPGuardError {
	errArr := make([]*standardNPGuardError, len(input))
	for i, err := range input {
		errArr[i] = &standardNPGuardError{
			err:      err.Error(),
			location: err.Location(),
			isSevere: err.IsSevere(),
		}
	}
	return errArr
}

func ConvertFileProcessingError(input []npganalyzer.FileProcessingError) []*standardNPGuardError {
	errArr := make([]*standardNPGuardError, len(input))
	for i, err := range input {
		errArr[i] = &standardNPGuardError{
			err:      err.Error(),
			location: err.Location(),
			isSevere: err.IsSevere(),
		}
	}
	return errArr
}

// HandleNPGerrors classifies NP-Guard errors as warnings or errors and ensures proper error location display
func HandleNPGerrors(src []*standardNPGuardError) (warns []error, errs []error) {
	for _, err := range src {
		e := err.Error()
		if err.Location() != "" {
			e = errors.Errorf("%s (at %q)", err.Error(), err.Location())
		}
		if err.IsSevere() {
			errs = append(errs, e)
		} else {
			warns = append(warns, e)
		}
	}
	return warns, errs
}

func SummarizeErrors(warns []error, errs []error, treatWarningsAsErrors bool, logger logger.Logger) error {
	var errToReturn error
	if len(errs) > 0 {
		errToReturn = npg.ErrErrors
	} else if treatWarningsAsErrors && len(warns) > 0 {
		errToReturn = npg.ErrWarnings
	}
	for _, warn := range warns {
		logger.WarnfLn("%s", warn.Error())
	}
	for _, err := range errs {
		logger.ErrfLn("%s", err.Error())
	}
	return errToReturn
}
