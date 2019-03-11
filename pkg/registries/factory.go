package registries

import (
	"github.com/stackrox/rox/generated/storage"
	artifactoryFactory "github.com/stackrox/rox/pkg/registries/artifactory"
	azureFactory "github.com/stackrox/rox/pkg/registries/azure"
	dockerFactory "github.com/stackrox/rox/pkg/registries/docker"
	dtrFactory "github.com/stackrox/rox/pkg/registries/dtr"
	ecrFactory "github.com/stackrox/rox/pkg/registries/ecr"
	googleFactory "github.com/stackrox/rox/pkg/registries/google"
	nexusFactory "github.com/stackrox/rox/pkg/registries/nexus"
	quayFactory "github.com/stackrox/rox/pkg/registries/quay"
	tenableFactory "github.com/stackrox/rox/pkg/registries/tenable"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Creator is the func stub that defines how to instantiate an image registry.
type Creator func(scanner *storage.ImageIntegration) (types.ImageRegistry, error)

// Factory provides a centralized location for creating ImageScanner from v1.ImageIntegrations.
type Factory interface {
	CreateRegistry(source *storage.ImageIntegration) (types.ImageRegistry, error)
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

	ecrFactoryType, ecrFactoryCreator := ecrFactory.Creator()
	reg.creators[ecrFactoryType] = ecrFactoryCreator

	googleFactoryType, googleFactoryCreator := googleFactory.Creator()
	reg.creators[googleFactoryType] = googleFactoryCreator

	quayFactoryType, quayFactoryCreator := quayFactory.Creator()
	reg.creators[quayFactoryType] = quayFactoryCreator

	tenableFactoryType, tenableFactoryCreator := tenableFactory.Creator()
	reg.creators[tenableFactoryType] = tenableFactoryCreator

	nexusFactoryType, nexusFactoryCreator := nexusFactory.Creator()
	reg.creators[nexusFactoryType] = nexusFactoryCreator

	azureFactoryType, azureFactoryCreator := azureFactory.Creator()
	reg.creators[azureFactoryType] = azureFactoryCreator

	return reg
}
