package matcher

import (
	configDS "github.com/stackrox/rox/central/config/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once         sync.Once
	soleInstance PlatformMatcher
)

func initialize() {
	soleInstance = New(configDS.Singleton())
}

// Singleton returns the sole instance of the PlatformMatcher.
func Singleton() PlatformMatcher {
	once.Do(initialize)
	return soleInstance
}
