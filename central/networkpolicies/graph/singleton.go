package graph

import (
	"sync"

	"github.com/stackrox/rox/central/globaldb"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/store"
)

var (
	once sync.Once

	ge *evaluatorImpl
)

func initialize() {
	ge = newGraphEvaluator(namespaceDataStore.New(globaldb.GetGlobalDB()))
}

// Singleton provides the interface for non-service external interaction.
func Singleton() Evaluator {
	once.Do(initialize)
	return ge
}
