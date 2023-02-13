package flags

import (
	"github.com/spf13/cobra"
)

var (
	password string
)

// AddPassword adds the password flag to the base command.
func AddPassword(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&password, "password", "p", "", "Password for basic auth")
}

// Password returns the set password.
func Password() string {
	return password
}
