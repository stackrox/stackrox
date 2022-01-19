package registries

type factoryOption struct {
	creatorFuncs []creatorWrapper
}

type FactoryOption interface {
	apply(*factoryOption)
}

type factoryOptionFunc func(*factoryOption)

func (o factoryOptionFunc) apply(opt *factoryOption) {
	o(opt)
}

// WithRegistryCreators specifies which registries to add to the factory.
func WithRegistryCreators(creatorFuncs ...creatorWrapper) FactoryOption {
	return factoryOptionFunc(func(o *factoryOption) {
		o.creatorFuncs = creatorFuncs
	})
}
