package main

// engines.go holds the two seams of the selector. A deps value pairs a
// rules language behind engine.Engine with a linter behind
// validation.Validator; the commands and the GitHub Actions planner
// receive it, so they depend on the interfaces only and a test can hand
// in any pair without touching shared state. Replacing either
// implementation means passing a different value from main.

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/engine"
	"github.com/stackrox/rox/ci/select/internal/engine/regexengine"
	"github.com/stackrox/rox/ci/select/internal/engine/regexengine/rulesvalidation"
	"github.com/stackrox/rox/ci/select/internal/validation"
)

// deps is the selector's pair of pluggable seams.
type deps struct {
	engine    engine.Engine
	validator validation.Validator
}

// productionDeps wires today's rules language with its full check set.
func productionDeps() deps {
	return deps{
		engine:    regexengine.New(),
		validator: rulesvalidation.Full{},
	}
}

// readRules loads the rules document from disk.
func (d deps) readRules(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	return data, errors.Wrap(err, "read rules")
}

// lint runs the configured validator over the document without
// compiling anything, for the validate command.
func (d deps) lint(path string) ([]string, error) {
	data, err := d.readRules(path)
	if err != nil {
		return nil, err
	}
	// The error is the rendered report the CLI prints verbatim; wrapping
	// it would corrupt the finding lines.
	warnings, err := d.validator.Validate(data)
	return warnings, err //nolint:wrapcheck
}

// prepare validates and compiles the document for Resolve. The engine
// loads only documents the validator accepted, so selection fails closed
// exactly where validation does; the engine's own Compile rejects
// whatever it cannot evaluate, whichever validator ran.
func (d deps) prepare(path string) (engine.Rules, []string, error) {
	data, err := d.readRules(path)
	if err != nil {
		return nil, nil, err
	}
	warnings, err := d.validator.Validate(data)
	if err != nil {
		//nolint:wrapcheck // The error is the rendered report the CLI prints verbatim.
		return nil, warnings, err
	}
	compiled, compileErr := d.engine.Compile(data)
	return compiled, warnings, compileErr //nolint:wrapcheck // Compile errors are printed as one-line findings, not wrapped causes.
}
