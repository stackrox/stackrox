package main

import (
	"strings"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

// ResourceType of the resource, determined according to resource metadata, schema and join type.
//go:generate stringer -type=ResourceType
type ResourceType int

const (
	Unknown ResourceType = iota
	JoinTable
	PermissionChecker
	GloballyScoped
	DirectlyScoped
	IndirectlyScoped
)

func getResourceType(storageType string, schema *walker.Schema, permissionChecker bool, joinTable bool) ResourceType {
	if joinTable {
		return JoinTable
	}
	if permissionChecker {
		return PermissionChecker
	}
	resource := storageToResource(storageType)
	metadata := resourceMetadataFromString(resource)

	clusterIDExists := false
	namespaceExists := false
	for _, f := range schema.Fields {
		if strings.Contains(f.Search.FieldName, "Cluster ID") {
			clusterIDExists = true
		}
		if strings.Contains(f.Search.FieldName, "Namespace") {
			namespaceExists = true
		}
	}

	switch metadata.GetScope() {
	case permissions.GlobalScope:
		return GloballyScoped
	case permissions.NamespaceScope:
		if clusterIDExists && namespaceExists {
			return DirectlyScoped
		} else {
			return IndirectlyScoped
		}
	case permissions.ClusterScope:
		if clusterIDExists {
			return DirectlyScoped
		} else {
			return IndirectlyScoped
		}
	default:
		return Unknown
	}
}
