package flags

import (
	"time"

	"github.com/spf13/cobra"
)

const (
	// Set the timeout to an invalid duration in the initializer
	// to make sure that the timeout is set by the flag.
	// This will prevent incorrect use of this flag.
	sentinelInvalidDuration = -1
)

var (
	timeout time.Duration = sentinelInvalidDuration
)

// AddTimeoutWithDefault adds a timeout flag to the given command, with the given default.
func AddTimeoutWithDefault(c *cobra.Command, defaul time.Duration) {
	c.PersistentFlags().DurationVarP(&timeout, "timeout", "t", defaul, "timeout for API requests")
}

// AddTimeout adds a timeout flag to the given command, with the global default value.
func AddTimeout(c *cobra.Command) {
	AddTimeoutWithDefault(c, 10*time.Second)
}

// Timeout returns the set timeout.
func Timeout() time.Duration {
	// This is a programming error. You shouldn't use the timeout flag unless you've added it to your command!
	// This helps us fail explicitly instead of defaulting to a zero timeout and allowing people to think it worked.
	if timeout == time.Duration(sentinelInvalidDuration) {
		panic("timeout not set by flag")
	}
	return timeout
}
