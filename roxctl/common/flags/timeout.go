package flags

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const (
	timeoutFlagName = "timeout"
)

// AddTimeoutWithDefault adds a timeout flag to the given command with the given default.
func AddTimeoutWithDefault(c *cobra.Command, defaultDuration time.Duration) {
	c.PersistentFlags().DurationP(timeoutFlagName, "t", defaultDuration, "timeout for API requests; represents the maximum duration of a request")
}

// AddTimeout adds a timeout flag to the given command with the global default value.
func AddTimeout(c *cobra.Command) {
	AddTimeoutWithDefault(c, 1*time.Minute)
}

// Timeout returns the set timeout.
func Timeout(c *cobra.Command) time.Duration {
	duration, err := c.Flags().GetDuration(timeoutFlagName)
	if err == nil {
		return duration
	}

	duration, err = c.PersistentFlags().GetDuration(timeoutFlagName)
	if err == nil {
		return duration
	}
	// This is a programming error. You shouldn't use the timeout flag unless you've added it to your command!
	// This helps us fail explicitly instead of defaulting to a zero timeout and allowing people to think it worked.
	panic(fmt.Sprintf("command does not have a timeout flag: %v", err))
}
