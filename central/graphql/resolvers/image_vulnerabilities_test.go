package resolvers

import (
	"reflect"
	"testing"
)

func TestImageVulnerabilityResolverType(t *testing.T) {
	// The implementation type has five more methods than the interface it implements.
	//
	//    2 - ComponentId
	//   12 - FirstImageOccurrence
	//   14 - ID
	//   19 - ImageId
	//   32 - State
	//
	// This can break graphQL queries to the system until
	// https://github.com/graph-gophers/graphql-go/issues/763 is fixed.
	// t.Skip("Interface and implementation types do not have aligned method indices.")
	resolverInterface := reflect.TypeOf((*ImageVulnerabilityResolver)(nil)).Elem()
	resolverImplType := reflect.TypeOf((*imageCVEV2Resolver)(nil))

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}
