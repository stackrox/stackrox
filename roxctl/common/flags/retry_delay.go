package flags

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

const (
	retryDelayDefaultValue = 10 * time.Second
	retryDelayFlagName     = "retry-delay"
)

// AddRetryDelayWithDefault adds a retry delay flag to the given command with the given default.
func AddRetryDelayWithDefault(c *cobra.Command, retryDelay time.Duration) {
	c.PersistentFlags().Duration(retryDelayFlagName, retryDelay, "Delay between retry attempts.")
}

// AddRetryDelay adds a retry delay flag to the given command with the global default value.
func AddRetryDelay(c *cobra.Command) {
	AddRetryDelayWithDefault(c, retryDelayDefaultValue)
}

// RetryDelay returns the set retry delay.
func RetryDelay(c *cobra.Command) time.Duration {
	duration, err := c.Flags().GetDuration(retryDelayFlagName)
	if err == nil {
		return duration
	}

	// This is a programming error. You shouldn't use the retry delay flag unless you've added it to your command!
	// This helps us fail explicitly instead of defaulting to zero delay and allowing people to think it worked.
	panic(fmt.Sprintf("command does not have a retry delay flag: %v", err))
}
