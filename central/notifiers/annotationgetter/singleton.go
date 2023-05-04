package annotationgetter

import (
	"github.com/stackrox/rox/central/notifiers"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	resolver *datastoreAnnotationGetter
)

func initialize() {
	resolver = newAnnotationGetter()
}

// Singleton provides the interface for getting annotation values with a datastore backed implementation.
func Singleton() notifiers.AnnotationGetter {
	once.Do(initialize)
	return resolver
}
