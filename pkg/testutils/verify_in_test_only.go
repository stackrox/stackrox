package testutils

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"testing"
)

var (
	testArgRegex = regexp.MustCompile(`^-test\.v($|=)`)
)

// MustBeInTest verifies that the given testing.T is a valid testingT from a running test.
// It is essentially impossible to create a valid one outside a test since the
// struct has no exported fields. We use this for methods that are "testing only", to make
// sure that they do not get exercised outside tests.
func MustBeInTest(t testing.TB) {
	if v := flag.Lookup("test.v"); v != nil && v.Value.String() == "true" {
		return
	}
	for _, arg := range os.Args {
		if testArgRegex.MatchString(arg) {
			return
		}
	}
	if t == nil || t.Name() == "" {
		panic(fmt.Sprintf("invalid testing T: %+v", t))
	}
}
