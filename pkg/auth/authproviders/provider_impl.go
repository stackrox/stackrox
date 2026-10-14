package authproviders

import (
	"context"
	"maps"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/auth/user"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	backendCreationInterval     = 10 * time.Second
	asyncBackendCreationTimeout = 30 * time.Second
)

var (
	errProviderDisabled = errors.New("provider has been deleted or disabled")
)

var _ Provider = (*providerImpl)(nil)

// If you add new data fields to this class, make sure you make commensurate modifications
// to the cloneWithoutMutex and copyWithoutMutex functions below.
// Exceptions: applyCtx and applyMu are live-provider-only; never propagate to clones.
type providerImpl struct {
	mutex   sync.RWMutex
	applyMu sync.Mutex // serializes the full ApplyOptions operation; held during CreateBackend

	storedInfo     *storage.AuthProvider
	backendFactory BackendFactory

	backend                    Backend
	backendCreationDone        concurrency.ErrorSignal
	lastBackendCreationAttempt time.Time

	roleMapper        permissions.RoleMapper
	issuer            tokens.Issuer
	attributeVerifier user.AttributeVerifier

	doNotStore bool

	validateCallback func() error

	// applyCtx threads the caller's context to CreateBackend inside ApplyOptions.
	// Set only on clones; never stored on the live provider.
	applyCtx context.Context
}

// Accessor functions.
//////////////////////

func (p *providerImpl) ID() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.storedInfo.GetId()
}

func (p *providerImpl) Type() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.storedInfo.GetType()
}

func (p *providerImpl) Name() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.storedInfo.GetName()
}

func (p *providerImpl) Enabled() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.storedInfo.GetEnabled()
}

func (p *providerImpl) Active() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.storedInfo.GetActive()
}

func (p *providerImpl) StorageView() *storage.AuthProvider {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := p.storedInfo.CloneVT()
	if result == nil {
		result = &storage.AuthProvider{}
	}
	if p.backendFactory != nil {
		result.Config = p.backendFactory.RedactConfig(result.GetConfig())
	} else {
		result.Config = nil
	}
	return result
}

func (p *providerImpl) BackendFactory() BackendFactory {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.backendFactory
}

func (p *providerImpl) GetOrCreateBackend(ctx context.Context) (Backend, error) {
	backend, err := concurrency.WithRLock2(&p.mutex, func() (Backend, error) {
		return p.backend, p.backendCreationDone.Err()
	})

	if backend != nil && err == nil {
		return backend, nil
	}

	// Backend factories are not guaranteed to survive product upgrades or restarts.
	if p.backendFactory == nil {
		return nil, errox.InvariantViolation.CausedBy(
			"the backend for this authentication provider cannot be instantiated;" +
				" this is probably because of a recent upgrade or a configuration change")
	}

	doneErrSig := concurrency.WithLock1(&p.mutex, func() concurrency.ReadOnlyErrorSignal {
		doneErrSig := p.backendCreationDone.Snapshot()
		if time.Since(p.lastBackendCreationAttempt) < backendCreationInterval {
			return doneErrSig
		}

		// Calling reset on the default value of an ErrorSignal returns true
		// so this works even in the default case
		if p.backendCreationDone.Reset() {
			go p.createBackendAsync(p.backendFactory, p.storedInfo.GetId(), AllUIEndpoints(p.storedInfo), p.storedInfo.GetConfig(),
				p.storedInfo.GetClaimMappings())

			p.lastBackendCreationAttempt = time.Now()
			return p.backendCreationDone.Snapshot()
		}
		return doneErrSig
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-doneErrSig.Done():
	}

	if err := doneErrSig.Err(); err != nil {
		return nil, err
	}

	backend = concurrency.WithRLock1(&p.mutex, func() Backend {
		return p.backend
	})
	if backend == nil {
		return nil, utils.ShouldErr(errors.New("unexpected: backend was nil"))
	}
	return backend, nil
}

func (p *providerImpl) createBackendAsync(factory BackendFactory, id string, allUIEndpoints []string, config map[string]string, mappings map[string]string) {
	ctx, cancel := context.WithTimeout(context.Background(), asyncBackendCreationTimeout)
	defer cancel()

	backend, err := factory.CreateBackend(ctx, id, allUIEndpoints, config, mappings)
	if err != nil {
		backend = nil
	} else if backend == nil {
		err = errors.New("factory returned nil backend")
	}

	p.mutex.Lock()
	oldBackend := p.backend
	p.backend = backend
	enabled := p.storedInfo.GetEnabled()
	p.backendCreationDone.SignalWithError(err)
	p.mutex.Unlock()

	// Fire lifecycle hooks outside the mutex so provider accessors can safely
	// acquire the read lock from within hook implementations.
	if enabled {
		if oldBackend != nil {
			oldBackend.OnDisable(p)
		}
		if backend != nil {
			backend.OnEnable(p)
		}
	}
}

func (p *providerImpl) Backend() Backend {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.backend
}

func (p *providerImpl) RoleMapper() permissions.RoleMapper {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.roleMapper
}

func (p *providerImpl) Issuer() tokens.Issuer {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.issuer
}

func (p *providerImpl) AttributeVerifier() user.AttributeVerifier {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.attributeVerifier
}

// Modifier functions.
//////////////////////

func (p *providerImpl) Validate(ctx context.Context, claims *tokens.Claims) error {
	if !p.Enabled() {
		return errProviderDisabled
	}

	if err := validateTokenProviderUpdate(p.StorageView(), claims); err != nil {
		return errors.Wrap(err, "token issued prior to provider update cannot be used")
	}

	backend, err := p.GetOrCreateBackend(ctx)
	if err != nil {
		return errors.Wrap(err, "provider is unavailable")
	}

	return backend.Validate(ctx, claims)
}

// We must lock the provider when applying options to it.
func (p *providerImpl) ApplyOptions(options ...ProviderOption) error {
	p.applyMu.Lock()
	defer p.applyMu.Unlock()

	var (
		priorBackend        Backend
		staleBackend        Backend
		wasEnabled          bool
		needsBackendRebuild bool
		modifiedProvider    *providerImpl
	)

	// Apply options to a clone under the lock; CreateBackend is intentionally
	// deferred so we do not hold the mutex during a potentially slow network call.
	err := func() error {
		p.mutex.Lock()
		defer p.mutex.Unlock()

		wasEnabled = p.storedInfo.GetEnabled()
		priorBackend = p.backend

		modifiedProvider = cloneWithoutMutex(p)
		if err := applyOptions(modifiedProvider, options...); err != nil {
			return err
		}

		// If an option (e.g. WithConfig) changed the stored config, the existing
		// backend is stale and must be replaced.
		if modifiedProvider.backend != nil && modifiedProvider.backendFactory != nil &&
			!maps.Equal(modifiedProvider.storedInfo.GetConfig(), modifiedProvider.backend.Config()) {
			staleBackend = modifiedProvider.backend
			modifiedProvider.backend = nil
			needsBackendRebuild = true
			return nil
		}

		copyWithoutMutex(p, modifiedProvider)
		return nil
	}()
	if err != nil {
		return err
	}

	// Rebuild the backend outside the lock when config drift was detected.
	if needsBackendRebuild {
		baseCtx := modifiedProvider.applyCtx
		if baseCtx == nil {
			baseCtx = context.Background()
		}
		rebuildCtx, cancel := context.WithTimeout(baseCtx, asyncBackendCreationTimeout)
		defer cancel()
		newBackend, err := modifiedProvider.backendFactory.CreateBackend(
			rebuildCtx,
			modifiedProvider.storedInfo.GetId(),
			AllUIEndpoints(modifiedProvider.storedInfo),
			modifiedProvider.storedInfo.GetConfig(),
			modifiedProvider.storedInfo.GetClaimMappings(),
		)
		if err != nil {
			return errors.Wrap(err, "failed to rebuild backend after config change")
		}
		modifiedProvider.backend = newBackend
		modifiedProvider.storedInfo.Config = newBackend.Config()

		// Guard against a concurrent backend update (e.g. createBackendAsync)
		// that may have already replaced p.backend while we rebuilt outside the lock.
		if err := func() error {
			p.mutex.Lock()
			defer p.mutex.Unlock()
			if p.backend != priorBackend {
				return errors.New("backend was concurrently updated; retry the operation")
			}
			copyWithoutMutex(p, modifiedProvider)
			return nil
		}(); err != nil {
			return err
		}
	}

	// Fire lifecycle hooks outside the mutex so provider accessors can safely
	// acquire the read lock from within hook implementations.
	//
	// Use wasEnabled (pre-update) for OnDisable so we clean up stale backends
	// regardless of whether the provider is currently enabled.
	if staleBackend != nil && wasEnabled {
		staleBackend.OnDisable(p)
	}
	// Fire OnEnable only when ApplyOptions itself introduced the backend
	// (priorBackend == nil) — e.g. the WithBackendFromFactory late-registration
	// path — because updatedNoLock will not be called afterwards.
	if priorBackend == nil && modifiedProvider.backend != nil && modifiedProvider.storedInfo.GetEnabled() {
		modifiedProvider.backend.OnEnable(p)
	}

	return nil
}

func (p *providerImpl) MarkAsActive() error {
	if p.Active() || p.validateCallback == nil {
		return nil
	}
	return p.validateCallback()
}

func (p *providerImpl) MergeConfigInto(newCfg map[string]string) map[string]string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	if p.backendFactory == nil {
		return newCfg
	}

	return p.backendFactory.MergeConfig(newCfg, p.storedInfo.GetConfig())
}

// restoreStoredInfo reverts storedInfo to a prior snapshot under applyMu + mutex,
// keeping live and stored state consistent after a persistence failure.
func (p *providerImpl) restoreStoredInfo(prior *storage.AuthProvider) {
	p.applyMu.Lock()
	defer p.applyMu.Unlock()
	p.mutex.Lock()
	p.storedInfo = prior.CloneVT()
	p.mutex.Unlock()
}

// Does a deep copy of the proto field 'storedInfo' so that it can support nested message fields.
func cloneWithoutMutex(pr *providerImpl) *providerImpl {
	return &providerImpl{
		storedInfo:     pr.storedInfo.CloneVT(),
		backendFactory: pr.backendFactory,
		backend:        pr.backend,
		roleMapper:     pr.roleMapper,
		issuer:         pr.issuer,
	}
}

// No need to do a deep copy of the 'storedInfo' field here since the 'from' input was created with a deep copy.
func copyWithoutMutex(to *providerImpl, from *providerImpl) {
	to.storedInfo = from.storedInfo.CloneVT()
	to.backendFactory = from.backendFactory
	to.backend = from.backend
	to.roleMapper = from.roleMapper
	to.issuer = from.issuer
}
