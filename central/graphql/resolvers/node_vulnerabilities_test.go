package resolvers

import (
	"reflect"
	"testing"
)

func TestNodeVulnerabilityResolverType(t *testing.T) {
	// The implementation type has six more methods than the interface it implements.
	//
	//    6 - ID
	//   18 - Orphaned
	//   19 - OrphanedTime
	//   23 - SnoozeExpiry
	//   24 - SnoozeStart
	//   25 - Snoozed
	//
	// This can break graphQL queries to the system until
	// https://github.com/graph-gophers/graphql-go/issues/763 is fixed.
	t.Skip("Interface and implementation types do not have aligned method indices.")
	resolverInterface := reflect.TypeOf((*NodeVulnerabilityResolver)(nil)).Elem()
	resolverImplType := reflect.TypeOf((*nodeCVEResolver)(nil))

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}
