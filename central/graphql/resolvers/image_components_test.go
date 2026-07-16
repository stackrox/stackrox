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
	//
	// TODO(ROX-35654): Unskip this tests.
	t.Skip("Interface and implementation types do not have aligned method indices.")
	resolverInterface := reflect.TypeFor[ImageComponentResolver]()
	resolverImplType := reflect.TypeFor[*imageComponentV2Resolver]()

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}
