package resolvers

import (
	"reflect"
	"testing"
)

func TestNodeComponentResolverType(t *testing.T) {
	resolverInterface := reflect.TypeOf((*NodeComponentResolver)(nil)).Elem()
	resolverImplType := reflect.TypeOf((*nodeComponentResolver)(nil))

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}
