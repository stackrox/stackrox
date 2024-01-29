package loaders

import (
	"context"
	"errors"
	"reflect"

	"github.com/stackrox/rox/pkg/sync"
)

type dataLoaderContextKey struct{}

// GetLoader returns the loader for the specified type from the context if it is available.
func GetLoader(ctx context.Context, loaderType reflect.Type) (interface{}, error) {
	key := dataLoaderContextKey{}
	reg := ctx.Value(key)
	if reg == nil {
		return nil, errors.New("no loader context present")
	}
	lc, isLoaderContext := reg.(*loaderContext)
	if !isLoaderContext {
		return nil, errors.New("loader context key used for wrong object type")
	}
	return lc.getLoader(loaderType)
}

// HasLoaderContext checks if given context has loaders registered
func HasLoaderContext(ctx context.Context) bool {
	key := dataLoaderContextKey{}
	reg := ctx.Value(key)
	return reg != nil
}

// WithLoaderContext returns a new context that is able to track loaders for registered types.
func WithLoaderContext(ctx context.Context) context.Context {
	key := dataLoaderContextKey{}
	reg := ctx.Value(key)
	if reg == nil {
		return context.WithValue(ctx, dataLoaderContextKey{}, newLoaderContext())
	}
	return ctx
}

// Object that holds all of the loaders in use on a context.
func newLoaderContext() *loaderContext {
	return &loaderContext{
		loaders: make(map[reflect.Type]interface{}),
	}
}

type loaderContext struct {
	lock    sync.RWMutex
	loaders map[reflect.Type]interface{}
}

func (lc *loaderContext) getLoader(lt reflect.Type) (interface{}, error) {
	loader := lc.readLoader(lt)
	if loader != nil {
		return loader, nil
	}
	return lc.createLoader(lt)
}

func (lc *loaderContext) readLoader(lt reflect.Type) interface{} {
	lc.lock.RLock()
	defer lc.lock.RUnlock()

	loader, isPresent := lc.loaders[lt]
	if !isPresent {
		return nil
	}
	return loader
}

func (lc *loaderContext) createLoader(lt reflect.Type) (interface{}, error) {
	lc.lock.Lock()
	defer lc.lock.Unlock()

	loader, isPresent := lc.loaders[lt]
	if !isPresent {
		var err error
		loader, err = Factory.CreateLoader(lt)
		if err != nil {
			return nil, err
		}
		lc.loaders[lt] = loader
	}
	return loader, nil
}
