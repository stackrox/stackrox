package scrape

import (
	"sync"

	"github.com/stackrox/rox/central/scrape/sensor/accept"
	"github.com/stackrox/rox/central/scrape/sensor/emit"
)

var (
	once sync.Once

	factory Factory
)

func initialize() {
	factory = NewFactory(emit.SingletonEmitter(), accept.SingletonAccepter())
}

// SingletonFactory provides the singleton instance of the controller for starting, getting, and killing scrapes.
func SingletonFactory() Factory {
	once.Do(initialize)
	return factory
}
