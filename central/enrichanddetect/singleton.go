package enrichanddetect

import (
	"github.com/stackrox/rox/central/detection/lifecycle"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	en EnricherAndDetector
)

func initialize() {
	en = New(lifecycle.SingletonManager())
}

// Singleton provides the singleton EnricherAndDetector to use.
func Singleton() EnricherAndDetector {
	once.Do(initialize)
	return en
}
