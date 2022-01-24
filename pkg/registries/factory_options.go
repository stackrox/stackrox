package registries

type factoryOption struct {
	creatorFuncs []creatorWrapper
}

type FactoryOption interface {
	apply(*factoryOption)
}

type factoryOptionFunc func(*factoryOption)

func (f factoryOptionFunc) apply(opt *factoryOption) {
	f(opt)
}

// WithRegistryCreators specifies which registries to add to the factory.
func WithRegistryCreators(creatorFuncs ...creatorWrapper) FactoryOption {
	return factoryOptionFunc(func(o *factoryOption) {
		o.creatorFuncs = append(o.creatorFuncs, creatorFuncs...)
	})
}
