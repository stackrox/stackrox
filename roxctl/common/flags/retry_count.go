package flags

import (
	"fmt"

	"github.com/spf13/cobra"
)

const (
	retryCountDefaultValue = 5
	retryCountFlagName     = "retries"
)

// AddRetryCountWithDefault adds a retry count flag to the given command with the given default.
func AddRetryCountWithDefault(c *cobra.Command, retryCount int) {
	c.PersistentFlags().Int(retryCountFlagName, retryCount, "Number of retries.")
}

// AddRetryCount adds a retry count flag to the given command with the global default value.
func AddRetryCount(c *cobra.Command) {
	AddRetryCountWithDefault(c, retryCountDefaultValue)
}

// RetryCount returns the retry count.
func RetryCount(c *cobra.Command) int {
	retryCount, err := c.Flags().GetInt(retryCountFlagName)
	if err == nil {
		return retryCount
	}

	// This is a programming error. You shouldn't use the retry flag unless you've added it to your command!
	// This helps us fail explicitly instead of defaulting to zero retries and allowing people to think it worked.
	panic(fmt.Sprintf("command does not have a retry count flag: %v", err))
}
