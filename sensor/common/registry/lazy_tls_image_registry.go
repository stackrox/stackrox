package registry

import (
	"context"
	"errors"
	"net/http"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

// lazyTLSCheckRegistry is a wrapper around an ImageRegistry that
// will perform a TLS check on first Metadata invocation.
type lazyTLSCheckRegistry struct {
	// source, dataSource, etc. mirror post initialization fields
	// from pkg/registries/docker.  These fields should be provided
	// during construction and are assumed to be valid.
	source           *storage.ImageIntegration
	dataSource       *storage.DataSource
	dockerConfig     *storage.DockerConfig
	url              string
	registryHostname string

	// creater creates the underlying registry from source during
	// initialization.
	creator types.Creator

	// registry is populated after successful initialization.
	registry types.Registry

	// tlsCheckCache is used to perform and cache the TLS checks.
	tlsCheckCache *tlsCheckCacheImpl

	// initialized tracks whether lazy initialization has completed.
	initialized      bool
	initializedMutex sync.Mutex

	// initError holds the most recent initializatino error.
	initError error
}

var _ types.ImageRegistry = (*lazyTLSCheckRegistry)(nil)

// Config will NOT trigger an initialization, however after successful
// initialization the values may change.
func (l *lazyTLSCheckRegistry) Config() *types.Config {
	if l.registry != nil {
		return l.registry.Config()
	}

	return &types.Config{
		Username:         l.dockerConfig.GetUsername(),
		Password:         l.dockerConfig.GetPassword(),
		Insecure:         l.dockerConfig.GetInsecure(),
		URL:              l.url,
		RegistryHostname: l.registryHostname,
	}
}

func (l *lazyTLSCheckRegistry) DataSource() *storage.DataSource {
	return l.dataSource
}

func (l *lazyTLSCheckRegistry) HTTPClient() *http.Client {
	utils.Should(errors.New("not implemented"))
	return nil
}

func (l *lazyTLSCheckRegistry) Match(image *storage.ImageName) bool {
	return urlfmt.TrimHTTPPrefixes(l.registryHostname) == image.GetRegistry()
}

func (l *lazyTLSCheckRegistry) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	l.lazyInit()
	if l.initError != nil {
		return nil, l.initError
	}

	return l.registry.Metadata(image)
}

func (l *lazyTLSCheckRegistry) Name() string {
	return l.source.GetName()
}

func (l *lazyTLSCheckRegistry) Source() *storage.ImageIntegration {
	return l.source
}

func (l *lazyTLSCheckRegistry) Test() error {
	utils.Should(errors.New("not implemented"))
	return nil
}

func (l *lazyTLSCheckRegistry) lazyInit() {
	if l.initialized {
		return
	}

	l.initializedMutex.Lock()
	defer l.initializedMutex.Unlock()

	// Short-circuit if another goroutine completed initialization.
	if l.initialized {
		return
	}

	// Do the TLS check.
	secure, skip, err := l.tlsCheckCache.CheckTLS(context.Background(), l.dockerConfig.GetEndpoint())
	if err != nil {
		log.Warnf("Lazy TLS check failed for %q (%s): %v", l.source.GetName(), l.source.GetId(), err)
		l.initError = err
		return
	}

	if skip {
		l.initError = errors.New("tls check skipped due to previous TLS check errors")
		return
	}

	// If we got here assume that initialization has completed, any errors encountered
	// are no longer temporal so do not try to repeat initialization.
	l.initialized = true

	// Update the underlying config (also updates source due to being a reference).
	l.dockerConfig.Insecure = !secure

	// Create the registry.
	l.registry, l.initError = l.creator(l.source)
	if l.initError != nil {
		log.Warnf("Lazy init failed for %q (%s), secure: %t: %v", l.source.GetName(), l.source.GetId(), secure, l.initError)
	} else {
		log.Debugf("Lazy init success for %q (%s), secure: %t", l.source.GetName(), l.source.GetId(), secure)
	}
}
