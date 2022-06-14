package main

import (
	"strings"

	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres/walker"
)

// ResourceType of the resource, determined according to resource metadata, schema and join type.
type ResourceType int

const (
	unknown ResourceType = iota
	joinTable
	permissionChecker
	globallyScoped
	directlyScoped
	indirectlyScoped
)

func getResourceType(storageType string, schema *walker.Schema, permissionCheckerEnabled bool, isJoinTable bool) ResourceType {
	if isJoinTable {
		return joinTable
	}
	if permissionCheckerEnabled {
		return permissionChecker
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
		return globallyScoped
	case permissions.NamespaceScope:
		if clusterIDExists && namespaceExists {
			return directlyScoped
		}
		return indirectlyScoped
	case permissions.ClusterScope:
		if clusterIDExists {
			return directlyScoped
		}
		return indirectlyScoped
	}

	return unknown
}
