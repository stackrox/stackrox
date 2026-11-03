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
	auth                  authn.Authenticator
	platform              v1.Platform
	insecureSkipTLSVerify bool
	force                 bool
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

func InsecureSkipTLSVerify(insecure bool) Option {
	return func(o *options) {
		o.insecureSkipTLSVerify = insecure
	}
}

// Force, when true, discards any previously stored ClairCore manifest/index-report/
// scanned-state for this exact manifest before indexing, so IndexContainerImage
// performs a genuine from-scratch re-index instead of reusing ClairCore's own
// per-manifest cache (which is keyed on image content and the configured scanner
// set, not on indexer configuration such as feature flags).
func Force(force bool) Option {
	return func(o *options) {
		o.force = force
	}
}

// ResolveForce reports the force setting that the given Options resolve to.
// Exported so callers in other packages (e.g. scanner/services tests) that
// only hold a []Option can verify which value was actually set, since the
// options struct itself is unexported.
func ResolveForce(opts ...Option) bool {
	return makeOptions(opts...).force
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
