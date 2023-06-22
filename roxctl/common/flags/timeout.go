package flags

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const (
	timeoutFlagName = "timeout"
)

// AddTimeoutWithDefault adds a timeout flag to the given command, with the given default.
func AddTimeoutWithDefault(c *cobra.Command, defaultDuration time.Duration) {
	c.PersistentFlags().DurationP(timeoutFlagName, "t", defaultDuration, "timeout for API requests")
}

// AddTimeout adds a timeout flag to the given command, with the global default value.
func AddTimeout(c *cobra.Command) {
	AddTimeoutWithDefault(c, 1*time.Minute)
}

// Timeout returns the set timeout.
func Timeout(c *cobra.Command) time.Duration {
	// Since a command can be the one adding the timeout flag via AddTimeout, make sure we look at the combined
	// list of persistent flag set and the commands flag set called LocalFlags.
	duration, err := c.LocalFlags().GetDuration(timeoutFlagName)
	if err != nil {
		// This is a programming error. You shouldn't use the timeout flag unless you've added it to your command!
		// This helps us fail explicitly instead of defaulting to a zero timeout and allowing people to think it worked.
		panic(fmt.Sprintf("command does not have a timeout flag: %v", err))
	}
	return duration
}
