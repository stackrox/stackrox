package main

// engines.go holds the two seams of the selector. rulesEngine owns the
// rules language behind the engine.Engine interface; rulesValidator
// lints documents behind the validation.Validator interface. Replacing
// either means adding an implementation and pointing one of these
// variables at it — callers depend on the interfaces only.

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/engine"
	"github.com/stackrox/rox/ci/select/internal/engine/regexengine"
	"github.com/stackrox/rox/ci/select/internal/validation"
)

var (
	rulesEngine    engine.Engine        = regexengine.New()
	rulesValidator validation.Validator = validation.Full{}
)

// readRules loads the rules document from disk.
func readRules(rulesPath string) ([]byte, error) {
	data, err := os.ReadFile(rulesPath)
	return data, errors.Wrap(err, "read rules")
}

// lintRules runs the selected validator over the document without
// compiling anything, for the validate command.
func lintRules(rulesPath string) ([]string, error) {
	data, err := readRules(rulesPath)
	if err != nil {
		return nil, err
	}
	// The error is the rendered report the CLI prints verbatim; wrapping
	// it would corrupt the finding lines.
	warnings, err := rulesValidator.Validate(data)
	return warnings, err //nolint:wrapcheck
}

// prepareRules validates and compiles the document for Resolve. The
// engine loads only documents the validator accepted, so selection
// fails closed exactly where validation does; the engine's own Compile
// rejects whatever it cannot evaluate, whichever validator ran.
func prepareRules(rulesPath string) (engine.Rules, []string, error) {
	data, err := readRules(rulesPath)
	if err != nil {
		return nil, nil, err
	}
	warnings, err := rulesValidator.Validate(data)
	if err != nil {
		//nolint:wrapcheck // The error is the rendered report the CLI prints verbatim.
		return nil, warnings, err
	}
	compiled, compileErr := rulesEngine.Compile(data)
	return compiled, warnings, compileErr //nolint:wrapcheck // Compile errors are printed as one-line findings, not wrapped causes.
}
