package ranking

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	clusterOnce   sync.Once
	clusterRanker *Ranker

	namespaceOnce   sync.Once
	namespaceRanker *Ranker

	deploymentOnce   sync.Once
	deploymentRanker *Ranker

	imageOnce   sync.Once
	imageRanker *Ranker

	imageComponentOnce   sync.Once
	imageComponentRanker *Ranker
)

// ClusterRanker returns the instance of ranker that ranks clusters.
func ClusterRanker() *Ranker {
	clusterOnce.Do(func() {
		clusterRanker = NewRanker()
	})
	return clusterRanker
}

// NamespaceRanker returns the instance of ranker that ranks namespaces.
func NamespaceRanker() *Ranker {
	namespaceOnce.Do(func() {
		namespaceRanker = NewRanker()
	})
	return namespaceRanker
}

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
