package main

// rulesfile.go loads, lints, and compiles the rules document. The
// engine carries its own checks, so these helpers take the engine and
// never name a concrete language: callers pass whichever engine they
// were given, and tests pass one configured differently.

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/ci/select/internal/engine"
)

// readRules loads the rules document from disk.
func readRules(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	return data, errors.Wrap(err, "read rules")
}

// lintRules runs the engine's checks over the document without compiling
// anything, for the validate command.
func lintRules(eng engine.Engine, path string) ([]string, error) {
	data, err := readRules(path)
	if err != nil {
		return nil, err
	}
	// The error is the rendered report the CLI prints verbatim; wrapping
	// it would corrupt the finding lines.
	warnings, err := eng.Validate(data)
	return warnings, err //nolint:wrapcheck
}

// prepareRules validates and compiles the document for Resolve. The
// engine loads only documents its own checks accepted, so selection fails
// closed exactly where validation does; Compile rejects whatever it
// cannot evaluate, whichever checks the language is configured with.
func prepareRules(eng engine.Engine, path string) (engine.Rules, []string, error) {
	data, err := readRules(path)
	if err != nil {
		return nil, nil, err
	}
	warnings, err := eng.Validate(data)
	if err != nil {
		//nolint:wrapcheck // The error is the rendered report the CLI prints verbatim.
		return nil, warnings, err
	}
	compiled, compileErr := eng.Compile(data)
	return compiled, warnings, compileErr //nolint:wrapcheck // Compile errors are printed as one-line findings, not wrapped causes.
}
