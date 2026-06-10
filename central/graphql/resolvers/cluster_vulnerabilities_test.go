package resolvers

import (
	"reflect"
	"testing"
)

func TestClusterVulnerabilityResolverType(t *testing.T) {
	// The implementation type has four more methods than the interface it implements.
	//
	//   17 - SnoozeExpiry
	//   18 - SnoozeStart
	//   19 - Snoozed
	//   24 - Type
	//
	// This can break graphQL queries to the system until
	// https://github.com/graph-gophers/graphql-go/issues/763 is fixed.
	//
	// TODO(ROX-35654): Unskip this tests.
	t.Skip("Interface and implementation types do not have aligned method indices.")
	resolverInterface := reflect.TypeFor[ClusterVulnerabilityResolver]()
	resolverImplType := reflect.TypeFor[*clusterCVEResolver]()

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}
