package registries

import "github.com/stackrox/rox/pkg/registries/types"

// FactoryOptions specifies optional configuration parameters for a registry factory.
type FactoryOptions struct {
	// CreatorFuncs specifies which registries to add to the factory.
	// By default, AllCreatorFuncs is used.
	CreatorFuncs []types.CreatorWrapper

	// CreateFuncsWithoutRepoList specifies registries to add to the factory
	// that do not make use of a repo list (`/v2/_catalog`) in matching
	// decisions.
	CreatorFuncsWithoutRepoList []types.CreatorWrapper
}
