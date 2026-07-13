package resolvers

import (
	"reflect"
	"testing"
)

func TestNodeComponentResolverType(t *testing.T) {
	resolverInterface := reflect.TypeFor[NodeComponentResolver]()
	resolverImplType := reflect.TypeFor[*nodeComponentResolver]()

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}
