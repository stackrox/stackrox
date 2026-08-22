package detector

import "github.com/stackrox/rox/pkg/sync"

var (
	once sync.Once
	d    Detector
)

func initialize() {
	d = New()
}

// Singleton provides the Detector instance.
func Singleton() Detector {
	once.Do(initialize)
	return d
}
