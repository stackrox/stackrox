package aggregator

import "github.com/stackrox/stackrox/pkg/sync"

var (
	once       sync.Once
	aggregator ProcessAggregator
)

func initialize() {
	aggregator = NewAggregator()
}

// Singleton returns a singleton instance of a process aggregator.
func Singleton() ProcessAggregator {
	once.Do(initialize)
	return aggregator
}
