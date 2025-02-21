package flags

import (
	"github.com/stackrox/rox/pkg/env"
)

var (
	password        string
	passwordChanged *bool
)

// Password returns the set password.
func Password() string {
	return flagOrSettingValue(password, *passwordChanged, env.PasswordEnv)
}

// PasswordChanged returns whether the password is provided as an argument.
func PasswordChanged() bool {
	return *passwordChanged
}
