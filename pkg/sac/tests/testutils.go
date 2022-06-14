package tests

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
)

func resourceWithAccess(access storage.Access, resource permissions.Resource) permissions.ResourceWithAccess {
	return permissions.ResourceWithAccess{
		Access: access,
		Resource: permissions.ResourceMetadata{
			Resource: resource,
		},
	}
}
