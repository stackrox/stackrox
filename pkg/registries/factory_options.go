package registries

// FactoryOptions specifies optional configuration parameters for a registry factory.
type FactoryOptions struct {
	// CreatorFuncs specifies which registries to add to the factory.
	// By default, AllCreatorFuncs is used.
	CreatorFuncs []CreatorWrapper
}
