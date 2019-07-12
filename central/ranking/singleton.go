package ranking

import "github.com/stackrox/rox/pkg/sync"

var (
	once   sync.Once
	ranker *Ranker
)

// DeploymentRanker returns the instance of ranker that ranks deployments.
func DeploymentRanker() *Ranker {
	once.Do(func() {
		ranker = NewRanker()
	})
	return ranker
}
