package roxmetadata

import (
	"sync"
)

var (
	singleton Metadata
	once      sync.Once
)

// Singleton returns the singleton instance to use.
func Singleton() Metadata {
	once.Do(func() {
		singleton = New()
	})
	return singleton
}
