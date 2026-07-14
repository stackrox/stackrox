package resolvers

import (
	"reflect"
	"testing"
)

func TestImageResolverType(t *testing.T) {
	resolverInterface := reflect.TypeFor[ImageResolver]()
	resolverImplType := reflect.TypeFor[*imageResolver]()

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}

func TestImageV2ResolverType(t *testing.T) {
	resolverInterface := reflect.TypeFor[ImageResolver]()
	resolverImplType := reflect.TypeFor[*imageV2Resolver]()

	validateAlignedMethodIndex(t, resolverInterface, resolverImplType)
}
