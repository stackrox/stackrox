package logic

import "github.com/pkg/errors"

var (
	// ErrAssumptionViolation is a marker error that signals an unsatisfied assumption.
	ErrAssumptionViolation = errors.New("assumption violation")
)
