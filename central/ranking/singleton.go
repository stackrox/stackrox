package ranking

import "github.com/stackrox/rox/pkg/sync"

var (
	deploymentOnce   sync.Once
	deploymentRanker *Ranker

	imageOnce   sync.Once
	imageRanker *Ranker

	serviceAccOnce   sync.Once
	serviceAccRanker *Ranker
)

// DeploymentRanker returns the instance of ranker that ranks deployments.
func DeploymentRanker() *Ranker {
	deploymentOnce.Do(func() {
		deploymentRanker = NewRanker()
	})
	return deploymentRanker
}

// ImageRanker returns the instance of ranker that ranks images.
func ImageRanker() *Ranker {
	imageOnce.Do(func() {
		imageRanker = NewRanker()
	})
	return imageRanker
}

// ServiceAccRanker returns the instance of ranker that ranks service accounts.
func ServiceAccRanker() *Ranker {
	serviceAccOnce.Do(func() {
		serviceAccRanker = NewRanker()
	})
	return serviceAccRanker
}
