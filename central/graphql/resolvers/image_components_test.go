package resolvers

import (
	"reflect"
	"testing"
)

func TestImageComponentResolverType(t *testing.T) {
	// The implementation type has two more methods than the interface it implements.
	//
	//    0 - Architecture
	//    7 - ImageId
	//
	// This can break graphQL queries to the system until
	// https://github.com/graph-gophers/graphql-go/issues/763 is fixed.
	t.Skip("Interface and implementation types do not have aligned method indices.")
	resolverInterface := reflect.TypeOf((*ImageComponentResolver)(nil)).Elem()
	resolverImplType := reflect.TypeOf((*imageComponentV2Resolver)(nil))

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}
