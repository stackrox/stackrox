package mode

import "github.com/stackrox/rox/pkg/concurrency"

var (
	interactiveMode concurrency.Flag
)

// SetInteractiveMode indicates that roxctl is running in interactive mode.
func SetInteractiveMode() {
	interactiveMode.Set(true)
}
