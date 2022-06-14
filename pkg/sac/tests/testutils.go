package tests

import (
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/auth/permissions"
)

func resourceWithAccess(access storage.Access, resource permissions.Resource) permissions.ResourceWithAccess {
	return permissions.ResourceWithAccess{
		Access: access,
		Resource: permissions.ResourceMetadata{
			Resource: resource,
		},
	}
}
