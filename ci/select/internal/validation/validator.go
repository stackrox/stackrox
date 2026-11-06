package validation

// validator.go declares the seam: callers lint documents through
// Validator without depending on how deep the checks go.

// Validator lints one rules document: the same raw bytes an Engine
// would compile, checked for authoring errors before selection runs.
// An error fails selection; warnings do not. Implementations may
// couple to a concrete engine's model, because what counts as a rule
// defect depends on the language.
type Validator interface {
	// Validate checks one document and returns its report: rendered
	// warning lines plus the footer, and an error whose text carries
	// the numbered error findings.
	Validate(data []byte) (warnings []string, err error)
}

// Trivial is the no-op end of the validator seam: every document gets
// an empty report. The engine still rejects what it cannot evaluate,
// so selection stays fail-closed; only rule-quality findings are
// skipped.
type Trivial struct{}

func (Trivial) Validate(_ []byte) ([]string, error) {
	return nil, nil
}
