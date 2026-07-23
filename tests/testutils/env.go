package testutils

import (
	"fmt"
	"os"
	"testing"
)

func lookupEnv(names ...string) (map[string]string, error) {
	vals := make(map[string]string, len(names))
	for _, n := range names {
		v, ok := os.LookupEnv(n)
		if !ok {
			return nil, fmt.Errorf("env var %s not set", n)
		}
		vals[n] = v
	}
	return vals, nil
}

// EnvOrFail returns the values of the named environment variables.
// It calls t.Fatalf if any variable is unset or empty.
func EnvOrFail(t testing.TB, names ...string) map[string]string {
	t.Helper()
	vals, err := lookupEnv(names...)
	if err != nil {
		t.Fatalf("%v", err)
	}
	return vals
}

// EnvOrSkip returns the values of the named environment variables.
// It calls t.Skipf if any variable is unset or empty.
func EnvOrSkip(t testing.TB, names ...string) map[string]string {
	t.Helper()
	vals, err := lookupEnv(names...)
	if err != nil {
		t.Skipf("%v", err)
	}
	return vals
}
