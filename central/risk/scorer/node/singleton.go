package node

import (
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once   sync.Once
	scorer Scorer
)

// GetScorer returns the singleton Scorer object to use when scoring risk.
func GetScorer() Scorer {
	once.Do(func() {
		scorer = NewNodeScorer()
	})
	return scorer
}
