package registry

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/uuid"
)

// lazyTLSCheckRegistry is a wrapper around a registry that performs
// TLS checks on first Metadata invocation.
type lazyTLSCheckRegistry struct {
	id   string
	name string
	// source, dataSource, etc. mirror post initialization fields from
	// pkg/registries/docker/docker.go. These fields are provided
	// during construction and are assumed to be valid.
	source           *storage.ImageIntegration
	dataSource       *storage.DataSource
	dockerConfig     *storage.DockerConfig
	url              string
	registryHostname string

	// creater creates the underlying registry from source during
	// initialization.
	creator        types.Creator
	creatorOptions []types.CreatorOption

	// registry is populated after successful initialization.
	registry types.Registry

	// tlsCheckCache performs and caches registry TLS checks.
	tlsCheckCache *tlsCheckCacheImpl

	// initialized tracks whether lazy initialization has completed.
	initialized atomic.Uint32
	// initializedMutex sync.RWMutex
	initializedMutex sync.DaveRWMutex

	// initError holds the most recent initialization error.
	initError error
}

var _ types.ImageRegistry = (*lazyTLSCheckRegistry)(nil)

// Config will NOT trigger initialization, however after successful
// initialization the values may change.
func (l *lazyTLSCheckRegistry) Config(ctx context.Context) *types.Config {
	rid := uuid.NewV4().String()
	log.Debugf("Config %q (trace %q, span %q) start", l.name, l.id, rid)
	start := time.Now()
	defer func() {
		log.Debugf("Config %q (trace %q, span %q) took %v", l.name, l.id, rid, time.Since(start))
	}()

	// registry is modified while the write lock is held,
	// to avoid a race grab the read lock.
	l.initializedMutex.RLock(l.id, rid)
	defer l.initializedMutex.RUnlock()

	if l.registry != nil {
		return l.registry.Config(ctx)
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
	rid := uuid.NewV4().String()
	log.Debugf("DataSource %q (trace %q, span %q) start", l.name, l.id, rid)
	start := time.Now()
	defer func() {
		log.Debugf("DataSource %q (trace %q, span %q) took %v", l.name, l.id, rid, time.Since(start))
	}()
	return l.dataSource
}

func (l *lazyTLSCheckRegistry) HTTPClient() *http.Client {
	utils.Should(errors.New("not implemented"))
	return nil
}

func (l *lazyTLSCheckRegistry) Match(image *storage.ImageName) bool {
	rid := uuid.NewV4().String()
	log.Debugf("Match %q (trace %q, span %q) start", l.name, l.id, rid)
	start := time.Now()
	defer func() {
		log.Debugf("Match %q (trace %q, span %q) took %v", l.name, l.id, rid, time.Since(start))
	}()
	return urlfmt.TrimHTTPPrefixes(l.registryHostname) == image.GetRegistry()
}

func (l *lazyTLSCheckRegistry) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	rid := uuid.NewV4().String()
	log.Debugf("Metadata %q (trace %q, span %q) start", l.name, l.id, rid)
	start := time.Now()
	defer func() {
		log.Debugf("Metadata %q (trace %q, span %q) took %v", l.name, l.id, rid, time.Since(start))
	}()

	// Attempt initialization since Metadata interacts with the registry.
	l.lazyInit()

	// initError and registry are modified while the
	// write lock is held, to avoid a race grab the
	// read lock.
	l.initializedMutex.RLock(l.id, rid)
	defer l.initializedMutex.RUnlock()

	if l.initError != nil {
		return nil, l.initError
	}

	return l.registry.Metadata(image)
}

func (l *lazyTLSCheckRegistry) Name() string {
	rid := uuid.NewV4().String()
	log.Debugf("Name %q (trace %q, span %q) start", l.name, l.id, rid)
	start := time.Now()
	defer func() {
		log.Debugf("Name %q (trace %q, span %q) took %v", l.name, l.id, rid, time.Since(start))
	}()
	// source is modified while the write lock is held,
	// to avoid a race grab the read lock.
	l.initializedMutex.RLock(l.id, rid)
	defer l.initializedMutex.RUnlock()

	return l.source.GetName()
}

func (l *lazyTLSCheckRegistry) Source() *storage.ImageIntegration {
	rid := uuid.NewV4().String()
	log.Debugf("Source %q (trace %q, span %q) start", l.name, l.id, rid)
	start := time.Now()
	defer func() {
		log.Debugf("Source %q (trace %q, span %q) took %v", l.name, l.id, rid, time.Since(start))
	}()
	// source is modified while the write lock is held,
	// to avoid a race grab the read lock.
	l.initializedMutex.RLock(l.id, rid)
	defer l.initializedMutex.RUnlock()

	return l.source
}

func (l *lazyTLSCheckRegistry) Test() error {
	utils.Should(errors.New("not implemented"))
	return nil
}

// lazyInit attempts to lazily perform a TLS check and initialize the
// underlying registry.  The concurrency mechanisms are based of
// https://cs.opensource.google/go/go/+/refs/tags/go1.22.5:src/sync/once.go;l=48
func (l *lazyTLSCheckRegistry) lazyInit() {
	rid := uuid.NewV4().String()
	log.Debugf("lazyInit %q (trace %q, span %q) start", l.name, l.id, rid)
	start := time.Now()
	defer func() {
		log.Debugf("lazyInit %q (trace %q, span %q) outer took %v", l.name, l.id, rid, time.Since(start))
	}()

	if l.isInitialized() {
		log.Debugf("lazyInit %q (trace %q, span %q) PRE lock short-circuit", l.name, l.id, rid)
		return
	}

	l.initializedMutex.Lock(l.id, rid)
	defer l.initializedMutex.Unlock(l.id, rid)

	innerstart := time.Now()
	defer func() {
		log.Debugf("lazyInit %q (trace %q, span %q) inner took %v", l.name, l.id, rid, time.Since(innerstart))
	}()

	// Short-circuit if another goroutine completed initialization.
	if l.isInitialized() {
		log.Debugf("lazyInit %q (trace %q, span %q) post lock short-circuit", l.name, l.id, rid)
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
	defer l.initialized.Store(1)

	// Clone the source to prevent a race.
	newSrc := l.source.CloneVT()
	dockerCfg := newSrc.GetDocker()
	if dockerCfg == nil {
		l.initError = errors.New("docker config is nil")
		return
	}
	dockerCfg.Insecure = !secure
	l.source = newSrc

	// Create the registry.
	l.registry, l.initError = l.creator(l.source, l.creatorOptions...)
	if l.initError != nil {
		log.Warnf("Lazy init failed for %q (%s), secure: %t: %v", l.source.GetName(), l.source.GetId(), secure, l.initError)
	} else {
		log.Debugf("Lazy init success for %q (%s), secure: %t -- %q (trace %q, span %q)", l.source.GetName(), l.source.GetId(), secure, l.name, l.id, rid)
	}
}

func (l *lazyTLSCheckRegistry) isInitialized() bool {
	return l.initialized.Load() == 1
}
