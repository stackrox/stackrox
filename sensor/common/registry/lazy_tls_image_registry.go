package registry

import (
	"context"
	"errors"
	"net/http"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sync"
)

// LazyTLSCheckRegistry is a wrapper around ImageRegistry that
// will perform a TLS check on first use instead of
// expecting a TLS check to have been done prior.
type LazyTLSCheckRegistry struct {
	// source represents the registry to create, however
	// the insecure flag associated with the underlying config is
	// assumed to be uninitialized.
	source *storage.ImageIntegration

	// creater is responsible for creating the underlying registry
	// based on imageIntegration when ready to do so.
	creator types.Creator

	dataSource *storage.DataSource

	// once is used to control the lazy initialization of the underlying
	// registry
	once sync.Once

	// initError will be populated if initialization of the underlying
	// registry fails.  If non nil will be returned or will alter
	// the return of the implemented ImageRegistry methods.
	initError error

	// checkTLSFunc is the function used to lazily check the TLS support
	// of the registry.
	tlsCheckCache *tlsCheckCacheImpl

	// registry is the underlying registry that is only populated after
	// successful lazy initialization.
	registry types.Registry
}

var _ types.ImageRegistry = (*LazyTLSCheckRegistry)(nil)

func (l *LazyTLSCheckRegistry) Config() *types.Config {
	// TODO: Find better way (in registry store) to determine if a hostname matches so
	// that initialization is NOT performed here and can be deferred to the first Metadata
	// call (ideal).
	// May have to dupe this logic:
	// https://github.com/stackrox/stackrox/blob/cb378f5d1341323c887d07868598d64a8a82d214/pkg/registries/docker/docker.go#L79
	l.lazyInit()
	if l.initError != nil {
		log.Debugf("Returning nil config for %q due to init error: %v", l.source.GetId(), l.initError)
		return nil
	}

	return l.registry.Config()
}

func (l *LazyTLSCheckRegistry) DataSource() *storage.DataSource {
	return l.dataSource
}

func (l *LazyTLSCheckRegistry) HTTPClient() *http.Client {
	l.lazyInit()
	if l.initError != nil {
		log.Debugf("Returning nil http client for %q due to init error: %v", l.source.GetId(), l.initError)
		return nil
	}

	return l.registry.HTTPClient()
}

func (l *LazyTLSCheckRegistry) Match(image *storage.ImageName) bool {
	l.lazyInit()
	if l.initError != nil {
		return false
	}

	return l.registry.Match(image)
}

func (l *LazyTLSCheckRegistry) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	l.lazyInit()
	if l.initError != nil {
		return nil, l.initError
	}

	return l.registry.Metadata(image)
}

func (l *LazyTLSCheckRegistry) Name() string {
	return l.source.GetName()
}

func (l *LazyTLSCheckRegistry) Source() *storage.ImageIntegration {
	l.lazyInit()
	return l.source
}

func (l *LazyTLSCheckRegistry) Test() error {
	l.lazyInit()
	if l.initError != nil {
		return l.initError
	}

	return l.registry.Test()
}

func (l *LazyTLSCheckRegistry) lazyInit() {
	l.once.Do(func() {
		log.Debugf("Lazily initializing registry %q (%s)", l.source.GetName(), l.source.GetId())

		// Get the registry endpoint.
		dockerCfg, err := extractDockerConfig(l.source)
		if err != nil {
			l.initError = err
			return
		}

		// Do the TLS check.
		secure, skip, err := l.tlsCheckCache.CheckTLS(context.Background(), dockerCfg.GetEndpoint())
		if err != nil {
			l.initError = err
			return
		}

		if skip {
			l.initError = errors.New("tls check skipped due to previous TLS check errors")
			return
		}

		// Update the underlying config.
		dockerCfg.Insecure = !secure

		// Create the registry.
		l.registry, err = l.creator(l.source)
		if err != nil {
			l.initError = err
			return
		}
	})
}

func extractDockerConfig(ii *storage.ImageIntegration) (*storage.DockerConfig, error) {
	if ii == nil {
		return nil, errors.New("nil image integration")
	}

	protoCfg := ii.GetIntegrationConfig()
	if protoCfg == nil {
		return nil, errors.New("nil image integration config")
	}

	cfg, ok := protoCfg.(*storage.ImageIntegration_Docker)
	if !ok || cfg == nil {
		return nil, errors.New("nil docker config")
	}

	return cfg.Docker, nil
}
