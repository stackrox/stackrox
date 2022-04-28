package common

import "github.com/stackrox/rox/pkg/errox"

var (
	// ErrInvalidCommandOption indicates bad options provided by the user
	// during invocation of roxctl command.
	ErrInvalidCommandOption = errox.InvalidArgs.New("invalid command option")

	// ErrDeprecatedFlag is error factory for commands with deprecated flags.
	ErrDeprecatedFlag = func(oldFlag, newFlag string) errox.Error {
		return errox.InvalidArgs.Newf("specified deprecated flag %q and new flag %q at the same time", oldFlag, newFlag)
	}
)
