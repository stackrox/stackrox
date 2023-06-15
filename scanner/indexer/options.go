package indexer

import (
	"runtime"

	"github.com/google/go-containerregistry/pkg/authn"
	v1 "github.com/google/go-containerregistry/pkg/v1"
)

// Option configures the options required to properly
// fetch the image from the registry.
type Option func(*options)

type options struct {
	auth     authn.Authenticator
	platform v1.Platform
}

// WithAuth specifies the authentication to use
// when reaching out to the registry.
//
// Default: authn.Anonymous
func WithAuth(auth authn.Authenticator) Option {
	return func(o *options) {
		o.auth = auth
	}
}

// defaultPlatform is the linux operating system
// and the running program's architecture.
var defaultPlatform = v1.Platform{
	Architecture: runtime.GOARCH,
	OS:           "linux", // We only support Linux containers at this time.
}

// WithPlatform specifies the desired OS and architecture of the image.
//
// Default: defaultPlatform.
func WithPlatform(platform v1.Platform) Option {
	return func(o *options) {
		o.platform = platform
	}
}

func makeOptions(opts ...Option) options {
	o := options{
		auth:     authn.Anonymous,
		platform: defaultPlatform,
	}

	for _, opt := range opts {
		opt(&o)
	}

	return o
}
