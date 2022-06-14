package main

import (
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

type object struct {
	storageType              string
	permissionCheckerEnabled bool
	isJoinTable              bool
	schema                   *walker.Schema
}

func (o object) GetID(name string) string {
	return identifierGetter(name, o.schema)
}

func (o object) GetClusterID(name string) string {
	return clusterGetter(name, o.schema)
}

func (o object) GetNamespace(name string) string {
	return namespaceGetter(name, o.schema)
}

func (o object) IsDirectlyScoped() bool {
	return o.isResourceType(directlyScoped)
}

func (o object) IsIndirectlyScoped() bool {
	return o.isResourceType(indirectlyScoped)
}

func (o object) IsGloballyScoped() bool {
	return o.isResourceType(globallyScoped)
}

func (o object) IsJoinTable() bool {
	return o.isJoinTable
}

func (o object) HasPermissionChecker() bool {
	return o.permissionCheckerEnabled
}

func (o object) IsNamespaceScope() bool {
	return o.isScope(permissions.NamespaceScope)
}

func (o object) IsClusterScope() bool {
	return o.isScope(permissions.ClusterScope)
}

func (o object) isScope(scope permissions.ResourceScope) bool {
	if o.isJoinTable || o.permissionCheckerEnabled {
		return false
	}
	resource := storageToResource(o.storageType)
	metadata := resourceMetadataFromString(resource)
	return metadata.GetScope() == scope
}

func (o object) isResourceType(resourceType ResourceType) bool {
	return getResourceType(o.storageType, o.schema, o.permissionCheckerEnabled, o.isJoinTable) == resourceType
}
