package registries

import dockerFactory "github.com/stackrox/rox/pkg/registries/docker"

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

// WithDockerRegistry adds the Docker registry creator to the registry factory.
func WithDockerRegistry() FactoryOption {
	return factoryOptionFunc(func(o *factoryOption) {
		o.creatorFuncs = append(o.creatorFuncs, dockerFactory.Creator)
	})
}
