package testutils

import (
	"os"
	"strconv"
)

// IsRunningInCI returns true if the process is invoked within a CI environment.
// This determination is made based on the existence of the 'CI' environment
// variable, unless when the value of this variable could be parsed as boolean
// false.
func IsRunningInCI() bool {
	v, set := os.LookupEnv("CI")
	if !set {
		return false
	}

	b, err := strconv.ParseBool(v)
	return err != nil || b
}
