package flags

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const (
	retryTimeoutFlagName = "retry-timeout"
)

// AddRetryTimeoutWithDefault adds a retry timeout flag to the given command with the given default.
func AddRetryTimeoutWithDefault(c *cobra.Command, defaultDuration time.Duration) {
	c.PersistentFlags().Duration(retryTimeoutFlagName, defaultDuration, "timeout after which API requests are retried; zero means the full request duration is awaited without retry")
}

// AddRetryTimeout adds a retry timeout flag to the given command with the global default value.
func AddRetryTimeout(c *cobra.Command) {
	AddRetryTimeoutWithDefault(c, 20*time.Second)
}

// RetryTimeout returns the set retry timeout.
func RetryTimeout(c *cobra.Command) time.Duration {
	duration, err := c.Flags().GetDuration(retryTimeoutFlagName)
	if err == nil {
		return duration
	}

	duration, err = c.PersistentFlags().GetDuration(timeoutFlagName)
	if err == nil {
		return duration
	}

	// This is a programming error. You shouldn't use the timeout flag unless you've added it to your command!
	// This helps us fail explicitly instead of defaulting to a zero timeout and allowing people to think it worked.
	panic(fmt.Sprintf("command does not have a retry timeout flag: %v", err))
}
