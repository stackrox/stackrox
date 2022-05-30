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

func resourceWithAccessAndReplacingResource(access storage.Access, resource permissions.Resource,
	replacingResource permissions.Resource) permissions.ResourceWithAccess {
	return permissions.ResourceWithAccess{
		Access: access,
		Resource: permissions.ResourceMetadata{
			Resource: resource,
			ReplacingResource: &permissions.ResourceMetadata{
				Resource: replacingResource,
			},
		},
	}
}
