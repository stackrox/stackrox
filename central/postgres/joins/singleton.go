package joins

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	inst Generator
)

func initialize() {
	inst = newJoinGenerator()
}

// Singleton returns the sole instance of the Generator that provides funtionality to retrive SQL join clauses.
func Singleton() Generator {
	once.Do(initialize)
	return inst
}
