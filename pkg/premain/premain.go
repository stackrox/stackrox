package premain

import "github.com/stackrox/rox/pkg/concurrency"

var (
	hasEnteredMain concurrency.Flag
)

// StartMain indicates that we have entered the program's main() function.
// This should be the first instruction in main() and is guaranteed to not block or panic.
func StartMain() {
	hasEnteredMain.Set(true)
}

// IsInPreMain checks whether we are still in the `pre-main()` phase of running the program (e.g., `init()` functions
// or evaluating RHSs of global variables).
func IsInPreMain() bool {
	return !hasEnteredMain.Get()
}
