package registries

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	artifactoryFactory "bitbucket.org/stack-rox/apollo/pkg/registries/artifactory"
	dockerFactory "bitbucket.org/stack-rox/apollo/pkg/registries/docker"
	dtrFactory "bitbucket.org/stack-rox/apollo/pkg/registries/dtr"
	googleFactory "bitbucket.org/stack-rox/apollo/pkg/registries/google"
	quayFactory "bitbucket.org/stack-rox/apollo/pkg/registries/quay"
	tenableFactory "bitbucket.org/stack-rox/apollo/pkg/registries/tenable"
	"bitbucket.org/stack-rox/apollo/pkg/registries/types"
)

// Creator is the func stub that defines how to instantiate an image registry.
type Creator func(scanner *v1.ImageIntegration) (types.ImageRegistry, error)

// Factory provides a centralized location for creating ImageScanner from v1.ImageIntegrations.
type Factory interface {
	CreateRegistry(source *v1.ImageIntegration) (types.ImageRegistry, error)
}

// NewFactory creates a new scanner factory.
func NewFactory() Factory {
	reg := &factoryImpl{
		creators: make(map[string]Creator),
	}

	// Add registries to factory.
	//////////////////////////////
	artifactoryFactoryType, artifactoryFactoryCreator := artifactoryFactory.Creator()
	reg.creators[artifactoryFactoryType] = artifactoryFactoryCreator

	dockerFactoryType, dockerFactoryCreator := dockerFactory.Creator()
	reg.creators[dockerFactoryType] = dockerFactoryCreator

	dtrFactoryType, dtrFactoryCreator := dtrFactory.Creator()
	reg.creators[dtrFactoryType] = dtrFactoryCreator

	googleFactoryType, googleFactoryCreator := googleFactory.Creator()
	reg.creators[googleFactoryType] = googleFactoryCreator

	quayFactoryType, quayFactoryCreator := quayFactory.Creator()
	reg.creators[quayFactoryType] = quayFactoryCreator

	tenableFactoryType, tenableFactoryCreator := tenableFactory.Creator()
	reg.creators[tenableFactoryType] = tenableFactoryCreator

	return reg
}
