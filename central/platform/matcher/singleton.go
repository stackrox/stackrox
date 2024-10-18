package matcher

import "github.com/stackrox/rox/pkg/sync"

var (
	once         sync.Once
	soleInstance PlatformMatcher
)

func initialize() {
	soleInstance = New()
}

// Singleton returns the sole instance of the PlatformMatcher.
func Singleton() PlatformMatcher {
	once.Do(initialize)
	return soleInstance
}
