package datagraph

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/secret/index"
	"bitbucket.org/stack-rox/apollo/central/secret/store"
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
