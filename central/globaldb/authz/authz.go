package authz

import (
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
)

// DBReadAccessAuthorizer returns an authorizer for checking that a user has permission to read the entire DB.
func DBReadAccessAuthorizer() authz.Authorizer {
	return user.With(resources.AllResourcesViewPermissions()...)
}

// DBWriteAccessAuthorizer returns an authorizer for checking that a user has permission to modify the entire DB.
func DBWriteAccessAuthorizer() authz.Authorizer {
	return user.With(resources.AllResourcesModifyPermissions()...)
}
