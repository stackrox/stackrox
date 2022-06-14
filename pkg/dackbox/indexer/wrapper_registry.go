package indexer

import (
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/pkg/sync"
)

// WrapperRegistry is a registry of all indexers we should use to store messages.
type WrapperRegistry interface {
	RegisterWrapper(prefix []byte, wrapper Wrapper)
	Matches(key []byte) bool

	Wrapper
}

// NewWrapperRegistry returns a new registry for the index.
func NewWrapperRegistry() WrapperRegistry {
	return &wrapperRegistryImpl{
		wrappers: make(map[string]Wrapper),
	}
}

type wrapperRegistryImpl struct {
	lock     sync.RWMutex
	wrappers map[string]Wrapper
}

func (ir *wrapperRegistryImpl) RegisterWrapper(prefix []byte, wrapper Wrapper) {
	ir.lock.Lock()
	defer ir.lock.Unlock()

	ir.wrappers[string(prefix)] = wrapper
}

func (ir *wrapperRegistryImpl) Matches(key []byte) bool {
	ir.lock.RLock()
	defer ir.lock.RUnlock()

	longestMatch := findLongestMatch(ir.wrappers, key)
	return longestMatch != nil
}

func (ir *wrapperRegistryImpl) Wrap(key []byte, msg proto.Message) (string, interface{}) {
	ir.lock.RLock()
	defer ir.lock.RUnlock()

	longestMatch := findLongestMatch(ir.wrappers, key)
	if longestMatch != nil {
		return longestMatch.Wrap(key, msg)
	}
	return "", nil
}
