package testutils

import (
	"os"
	"strconv"
)

// IsRunningInCI returns true if a test invocation happens in CI.
func IsRunningInCI() bool {
	v, set := os.LookupEnv("CI")
	if !set {
		return false
	}

	b, err := strconv.ParseBool(v)
	return err != nil || b
}
