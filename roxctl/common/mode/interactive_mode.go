package mode

import "github.com/stackrox/rox/pkg/concurrency"

var (
	interactiveMode concurrency.Flag
)

// IsInInteractiveMode checks if roxctl is running in interactive mode.
func IsInInteractiveMode() bool {
	return interactiveMode.Get()
}

// SetInteractiveMode indicates that roxctl is running in interactive mode.
func SetInteractiveMode() {
	interactiveMode.Set(true)
}
