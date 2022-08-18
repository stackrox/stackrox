package ranking

import (
	"github.com/stackrox/rox/pkg/sync"
)

var (
	clusterOnce   sync.Once
	clusterRanker *Ranker

	namespaceOnce   sync.Once
	namespaceRanker *Ranker

	nodeOnce   sync.Once
	nodeRanker *Ranker

	deploymentOnce   sync.Once
	deploymentRanker *Ranker

	imageOnce   sync.Once
	imageRanker *Ranker

	componentOnce   sync.Once
	componentRanker *Ranker

	nodeComponentOnce   sync.Once
	nodeComponentRanker *Ranker
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

// NodeRanker returns the instance of ranker that ranks nodes.
func NodeRanker() *Ranker {
	nodeOnce.Do(func() {
		nodeRanker = NewRanker()
	})
	return nodeRanker
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

// ComponentRanker returns the instance of ranker that ranks image components.
func ComponentRanker() *Ranker {
	componentOnce.Do(func() {
		componentRanker = NewRanker()
	})
	return componentRanker
}

// NodeComponentRanker returns the instance of ranker that ranks node components.
func NodeComponentRanker() *Ranker {
	nodeComponentOnce.Do(func() {
		nodeComponentRanker = NewRanker()
	})
	return nodeComponentRanker
}
