package engine

// validator.go holds the linting contract and the no-op implementation.
// The contract lives here, beside the language it checks, because
// whether a rules document is well-formed is defined by that language:
// two engines may accept entirely different YAML. An engine hands out
// its own checks through Engine.Validate, so a validator can never be
// paired with a language it cannot read.

// Validator lints one rules document: the same raw bytes an Engine
// compiles. An error fails selection; warnings do not. Each language
// supplies implementations that understand its own syntax.
type Validator interface {
	// Validate checks one document and returns its report: rendered
	// warning lines plus the footer, and an error whose text carries
	// the numbered error findings.
	Validate(data []byte) (warnings []string, err error)
}

// TrivialValidator checks nothing. Languages may offer it to disable
// authoring checks; compilation still rejects documents the engine
// cannot evaluate, so selection stays fail-closed.
type TrivialValidator struct{}

func (TrivialValidator) Validate(_ []byte) ([]string, error) {
	return nil, nil
}
