//go:build test

// This file exists so that the vm-scanning-unit-tests Makefile target (which
// runs `go test ./vmhelpers`) succeeds even before the stacked branch
// piotr/ROX-29577-vm-e2e-helper-unit-tests adds the full unit-test suite.
// Without at least one _test.go file, `go test` produces output that the
// shared `report` target cannot parse, failing the CI lane.

package vmhelpers

import "testing"

func TestPlaceholder(t *testing.T) {
	t.Log("vmhelpers package compiles; full unit tests live on the 'piotr/ROX-29577-vm-e2e-helper-unit-tests' branch")
}
