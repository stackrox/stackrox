package permissioncheck

import "github.com/pkg/errors"

var (
	// ErrPermissionCheckOnly should be returned by an authorizer method if a permission check
	// object was detected in the context.
	ErrPermissionCheckOnly = errors.New("performing permission check only")
)
