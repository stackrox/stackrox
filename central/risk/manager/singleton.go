package manager

import (
	acUpdater "github.com/stackrox/rox/central/activecomponent/updater"
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	nodeDS "github.com/stackrox/rox/central/node/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	componentScorer "github.com/stackrox/rox/central/risk/scorer/component/singleton"
	deploymentScorer "github.com/stackrox/rox/central/risk/scorer/deployment"
	imageScorer "github.com/stackrox/rox/central/risk/scorer/image"
	nodeScorer "github.com/stackrox/rox/central/risk/scorer/node"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	manager Manager
)

func initialize() {
	manager = New(nodeDS.Singleton(),
		deploymentDS.Singleton(),
		imageDS.Singleton(),
		riskDS.Singleton(),

		nodeScorer.GetScorer(),
		componentScorer.GetNodeScorer(),
		deploymentScorer.GetScorer(),
		imageScorer.GetScorer(),
		componentScorer.GetImageScorer(),

		ranking.ClusterRanker(),
		ranking.NamespaceRanker(),
		ranking.ComponentRanker(),
		ranking.NodeComponentRanker(),

		acUpdater.Singleton())
}

// Singleton provides the singleton Manager to use.
func Singleton() Manager {
	once.Do(initialize)
	return manager
}
