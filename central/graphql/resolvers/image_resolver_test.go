package resolvers

import (
	"reflect"
	"testing"
)

func TestImageResolverType(t *testing.T) {
	resolverInterface := reflect.TypeOf((*ImageResolver)(nil)).Elem()
	resolverImplType := reflect.TypeOf((*imageResolver)(nil))

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}

func TestImageV2ResolverType(t *testing.T) {
	resolverInterface := reflect.TypeOf((*ImageResolver)(nil)).Elem()
	resolverImplType := reflect.TypeOf((*imageV2Resolver)(nil))

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}
