package image

import (
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once   sync.Once
	scorer Scorer
)

func initialize() {
	scorer = NewImageScorer()
}

// GetScorer returns the singleton Scorer object to use when scoring risk.
func GetScorer() Scorer {
	once.Do(initialize)
	return scorer
}
