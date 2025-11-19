package keyfence

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	kf   concurrency.KeyFence
)

// ImageKeyFenceSingleton provides a key fence for image and its sub-components.
func ImageKeyFenceSingleton() concurrency.KeyFence {
	once.Do(func() {
		kf = concurrency.NewKeyFence()
	})
	return kf
}
