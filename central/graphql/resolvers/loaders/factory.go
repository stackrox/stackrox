package loaders

import (
	"reflect"

	"github.com/pkg/errors"
)

// Factory is a unified presentation of the ability to generate Loaders for different types.
var Factory = newLoaderFactory()

// TypedFactory is a Loader factory. It returns a new instance of a Loader for a single type every call.
type TypedFactory func() interface{}

// RegisterTypeFactory registers a TypedFactory for the given reflect.Type, which is the key we use to decide
// what kind of loader to generate.
func RegisterTypeFactory(lt reflect.Type, factory TypedFactory) {
	Factory.typedFactories[lt] = factory
}

func newLoaderFactory() *loaderFactoryImpl {
	return &loaderFactoryImpl{
		typedFactories: make(map[reflect.Type]TypedFactory),
	}
}

type loaderFactoryImpl struct {
	typedFactories map[reflect.Type]TypedFactory
}

// CreateLoader creates a new, type specific, loader for the given input type.
func (lf *loaderFactoryImpl) CreateLoader(lt reflect.Type) (interface{}, error) {
	loaderFactory, exists := lf.typedFactories[lt]
	if !exists {
		return nil, errors.New("do not have loader for type")
	}
	return loaderFactory(), nil
}
