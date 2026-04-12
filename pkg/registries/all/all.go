// Package all provides the complete set of registry creator functions, including
// those that require heavy cloud SDK dependencies (AWS, Azure, GCP).
// Import this package only in binaries that need all registry types (e.g. Central).
// Sensor and other lightweight binaries should construct their own creator lists
// from the individual registry packages they need.
package all

import (
	artifactoryFactory "github.com/stackrox/rox/pkg/registries/artifactory"
	artifactRegistryFactory "github.com/stackrox/rox/pkg/registries/artifactregistry"
	azureFactory "github.com/stackrox/rox/pkg/registries/azure"
	dockerFactory "github.com/stackrox/rox/pkg/registries/docker"
	ecrFactory "github.com/stackrox/rox/pkg/registries/ecr"
	ghcrFactory "github.com/stackrox/rox/pkg/registries/ghcr"
	googleFactory "github.com/stackrox/rox/pkg/registries/google"
	ibmFactory "github.com/stackrox/rox/pkg/registries/ibm"
	nexusFactory "github.com/stackrox/rox/pkg/registries/nexus"
	quayFactory "github.com/stackrox/rox/pkg/registries/quay"
	rhelFactory "github.com/stackrox/rox/pkg/registries/rhel"
	"github.com/stackrox/rox/pkg/registries/types"
)

// CreatorFuncs defines all known registry creators.
var CreatorFuncs = []types.CreatorWrapper{
	artifactRegistryFactory.Creator,
	artifactoryFactory.Creator,
	azureFactory.Creator,
	dockerFactory.Creator,
	ecrFactory.Creator,
	ghcrFactory.Creator,
	googleFactory.Creator,
	ibmFactory.Creator,
	nexusFactory.Creator,
	quayFactory.Creator,
	rhelFactory.Creator,
}

// CreatorFuncsWithoutRepoList defines all known registry creators with repo list disabled.
var CreatorFuncsWithoutRepoList = []types.CreatorWrapper{
	artifactRegistryFactory.CreatorWithoutRepoList,
	artifactoryFactory.CreatorWithoutRepoList,
	azureFactory.CreatorWithoutRepoList,
	dockerFactory.CreatorWithoutRepoList,
	ecrFactory.CreatorWithoutRepoList,
	ghcrFactory.CreatorWithoutRepoList,
	googleFactory.CreatorWithoutRepoList,
	ibmFactory.CreatorWithoutRepoList,
	nexusFactory.CreatorWithoutRepoList,
	quayFactory.CreatorWithoutRepoList,
	rhelFactory.CreatorWithoutRepoList,
}
