package auth

import "github.com/stackrox/rox/pkg/errox"

var (
	// ErrNoValidRole indicates that though user credentials have been
	// provided, they do not specify a valid role. This usually happens because
	// of misconfigured access control. The effect is similar to NoCredentials.
	ErrNoValidRole = errox.NoCredentials.New("access for this user is not authorized: no valid role," +
		" please contact your system administrator")
)
