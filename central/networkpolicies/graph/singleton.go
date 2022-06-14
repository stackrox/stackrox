package graph

import (
	namespaceDataStore "github.com/stackrox/stackrox/central/namespace/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once

	ge *evaluatorImpl
)

func initialize() {
	ge = newGraphEvaluator(namespaceDataStore.Singleton())
}

// Singleton provides the interface for non-service external interaction.
func Singleton() Evaluator {
	once.Do(initialize)
	return ge
}
