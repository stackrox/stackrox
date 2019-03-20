package processor

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/central/notifier/store"
)

var (
	once sync.Once

	pr Processor
)

func initialize() {
	var err error
	pr, err = New(store.Singleton())
	if err != nil {
		panic(err)
	}
	go pr.Start()
}

// Singleton provides the interface for processing notifications.
func Singleton() Processor {
	once.Do(initialize)
	return pr
}
