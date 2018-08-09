package datagraph

import (
	"sync"

	"github.com/stackrox/rox/central/secret/index"
	"github.com/stackrox/rox/central/secret/store"
)

var (
	once sync.Once

	dg DataGraph
)

func initialize() {
	dg = New(store.Singleton(), index.Singleton())
}

// Singleton provides the instance of the DataGraph interface to register.
func Singleton() DataGraph {
	once.Do(initialize)
	return dg
}
