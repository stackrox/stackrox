package ranking

import "github.com/stackrox/rox/pkg/sync"

var (
	deploymentOnce   sync.Once
	deploymentRanker *Ranker

	imageOnce   sync.Once
	imageRanker *Ranker

	imageComponentOnce   sync.Once
	imageComponentRanker *Ranker
)

// DeploymentRanker returns the instance of ranker that ranks deployments.
func DeploymentRanker() *Ranker {
	deploymentOnce.Do(func() {
		deploymentRanker = NewRanker()
	})
	return deploymentRanker
}

// ImageRanker returns the instance of ranker that ranks image.
func ImageRanker() *Ranker {
	imageOnce.Do(func() {
		imageRanker = NewRanker()
	})
	return imageRanker
}

// ImageComponentRanker returns the instance of ranker that ranks image components.
func ImageComponentRanker() *Ranker {
	imageComponentOnce.Do(func() {
		imageComponentRanker = NewRanker()
	})
	return imageComponentRanker
}
