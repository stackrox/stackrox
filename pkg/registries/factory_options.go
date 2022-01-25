package registries

type factoryOption struct {
	creatorFuncs []creatorWrapper
}

// FactoryOption specifies optional configuration parameters for a registry factory.
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
