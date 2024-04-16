package testutils

import (
	"os"
)

// IsRunningInCI returns true if the process is invoked within a CI environment.
// This determination is made based on the existence of the 'CI' environment
// variable. The actual value of the variable is ignored.
func IsRunningInCI() bool {
	_, set := os.LookupEnv("CI")
	return set
}
