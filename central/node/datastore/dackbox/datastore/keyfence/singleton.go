package keyfence

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	kf concurrency.KeyFence
)

// NodeKeyFenceSingleton provides a key fence for node and its sub-components.
func NodeKeyFenceSingleton() concurrency.KeyFence {
	once.Do(func() {
		kf = concurrency.NewKeyFence()
	})
	return kf
}
