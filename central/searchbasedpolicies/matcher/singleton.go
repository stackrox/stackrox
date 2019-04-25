package matcher

import (
	deploymentMappings "github.com/stackrox/rox/central/deployment/index/mappings"
	imageMappings "github.com/stackrox/rox/central/image/index/mappings"
	"github.com/stackrox/rox/central/processindicator/datastore"
	"github.com/stackrox/rox/central/searchbasedpolicies/fields"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	registry          fields.Registry
	deploymentBuilder Builder
	imageBuilder      Builder
)

func intialize() {
	registry = fields.NewRegistry(datastore.Singleton())
	deploymentBuilder = NewBuilder(registry, deploymentMappings.OptionsMap)
	imageBuilder = NewBuilder(registry, imageMappings.OptionsMap)
}

// RegistrySingleton returns the registry used by the singleton matcher builders.
func RegistrySingleton() fields.Registry {
	once.Do(intialize)
	return registry
}

// DeploymentBuilderSingleton Builder when you want to build Matchers for deployment policies.
func DeploymentBuilderSingleton() Builder {
	once.Do(intialize)
	return deploymentBuilder
}

// ImageBuilderSingleton Builder when you want to build Matchers for image policies.
func ImageBuilderSingleton() Builder {
	once.Do(intialize)
	return imageBuilder
}
