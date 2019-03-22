package graph

import (
	"github.com/stackrox/rox/central/globaldb"
	namespaceDataStore "github.com/stackrox/rox/central/namespace/store"
	"github.com/stackrox/rox/pkg/sync"
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
