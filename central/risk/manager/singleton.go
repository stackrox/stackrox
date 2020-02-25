package manager

import (
	deploymentDS "github.com/stackrox/rox/central/deployment/datastore"
	imageDS "github.com/stackrox/rox/central/image/datastore"
	imageComponentDS "github.com/stackrox/rox/central/imagecomponent/datastore"
	"github.com/stackrox/rox/central/ranking"
	riskDS "github.com/stackrox/rox/central/risk/datastore"
	deploymentScorer "github.com/stackrox/rox/central/risk/scorer/deployment"
	imageScorer "github.com/stackrox/rox/central/risk/scorer/image"
	imageComponentScorer "github.com/stackrox/rox/central/risk/scorer/image_component"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once    sync.Once
	manager Manager
)

func initialize() {
	var err error
	manager, err = New(deploymentDS.Singleton(),
		imageDS.Singleton(),
		imageComponentDS.Singleton(),
		riskDS.Singleton(),

		deploymentScorer.GetScorer(),
		imageScorer.GetScorer(),
		imageComponentScorer.GetScorer(),

		ranking.ClusterRanker(),
		ranking.NamespaceRanker(),
		ranking.DeploymentRanker(),
		ranking.ImageRanker(),
		ranking.ImageComponentRanker())
	if err != nil {
		panic(err)
	}
}

// Singleton provides the singleton Manager to use.
func Singleton() Manager {
	once.Do(initialize)
	return manager
}
