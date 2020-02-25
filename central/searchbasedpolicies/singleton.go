package searchbasedpolicies

import (
	processDataStore "github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/pkg/search/options/deployments"
	"github.com/stackrox/rox/pkg/search/options/images"
	"github.com/stackrox/rox/pkg/searchbasedpolicies/matcher"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	registry          matcher.Registry
	deploymentBuilder matcher.Builder
	imageBuilder      matcher.Builder
)

func initialize() {
	registry = matcher.NewRegistry(
		processDataStore.Singleton(),
	)
	deploymentBuilder = matcher.NewBuilder(registry, deployments.OptionsMap)
	imageBuilder = matcher.NewBuilder(registry, images.OptionsMap)
}

// RegistrySingleton returns the registry used by the singleton matcher builders.
func RegistrySingleton() matcher.Registry {
	once.Do(initialize)
	return registry
}

// DeploymentBuilderSingleton Builder when you want to build Matchers for deployment policies.
func DeploymentBuilderSingleton() matcher.Builder {
	once.Do(initialize)
	return deploymentBuilder
}

// ImageBuilderSingleton Builder when you want to build Matchers for image policies.
func ImageBuilderSingleton() matcher.Builder {
	once.Do(initialize)
	return imageBuilder
}
