package flags

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
)

var (
	password        string
	passwordChanged *bool
)

// AddPassword adds the password flag to the base command.
func AddPassword(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&password, "password", "p", "",
		"password for basic auth. Alternatively, set the password via the ROX_ADMIN_PASSWORD environment variable")
	passwordChanged = &c.PersistentFlags().Lookup("password").Changed
}

// Password returns the set password.
func Password() string {
	return flagOrSettingValue(password, *passwordChanged, env.PasswordEnv)
}
