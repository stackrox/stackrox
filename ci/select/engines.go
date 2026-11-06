package main

// engines.go holds the two seams of the selector. A deps value carries
// one rules language behind engine.Engine; that engine also supplies the
// checks for its own documents, so a validator cannot be paired with a
// language it cannot read. The commands and the GitHub Actions planner
// receive the value, so they depend on the interfaces only and a test can
// hand in any language without touching shared state. Replacing the
// language means passing a different deps from main.

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/engine"
	"github.com/stackrox/rox/ci/select/internal/engine/regexengine"
)

// deps is the selector's rules language and the checks it performs.
// The validator is not a separate field on purpose: whoever owns the
// syntax owns the definition of a well-formed document.
type deps struct {
	engine engine.Engine
}

// productionDeps wires today's rules language with its full check set.
func productionDeps() deps {
	return deps{engine: regexengine.New()}
}

// readRules loads the rules document from disk.
func (d deps) readRules(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	return data, errors.Wrap(err, "read rules")
}

// lint runs the engine's checks over the document without compiling
// anything, for the validate command.
func (d deps) lint(path string) ([]string, error) {
	data, err := d.readRules(path)
	if err != nil {
		return nil, err
	}
	// The error is the rendered report the CLI prints verbatim; wrapping
	// it would corrupt the finding lines.
	warnings, err := d.engine.Validate(data)
	return warnings, err //nolint:wrapcheck
}

// prepare validates and compiles the document for Resolve. The engine
// loads only documents its own checks accepted, so selection fails closed
// exactly where validation does; Compile rejects whatever it cannot
// evaluate, whichever checks the language is configured with.
func (d deps) prepare(path string) (engine.Rules, []string, error) {
	data, err := d.readRules(path)
	if err != nil {
		return nil, nil, err
	}
	warnings, err := d.engine.Validate(data)
	if err != nil {
		//nolint:wrapcheck // The error is the rendered report the CLI prints verbatim.
		return nil, warnings, err
	}
	compiled, compileErr := d.engine.Compile(data)
	return compiled, warnings, compileErr //nolint:wrapcheck // Compile errors are printed as one-line findings, not wrapped causes.
}
