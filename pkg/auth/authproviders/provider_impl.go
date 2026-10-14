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
type providerImpl struct {
	mutex sync.RWMutex

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

	// Assign p.backend before signalling backendCreationDone so that any waiter
	// unblocked by the signal observes a non-nil backend immediately.
	p.mutex.Lock()
	oldBackend := p.backend
	p.backend = backend
	enabled := p.storedInfo.GetEnabled()
	p.backendCreationDone.SignalWithError(err)
	p.mutex.Unlock()

	// Fire lifecycle hooks outside the mutex so that Provider accessor methods called
	// from within hooks can safely acquire the read lock.
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
	// priorBackend is the backend that existed before any options ran. It is
	// used later to decide whether ApplyOptions itself should fire OnEnable for
	// a newly introduced backend (e.g. the WithBackendFromFactory path, where
	// updatedNoLock is never called after ApplyOptions returns).
	var (
		priorBackend        Backend
		staleBackend        Backend
		wasEnabled          bool
		needsBackendRebuild bool
		modifiedProvider    *providerImpl
	)

	// Step 1: under the lock, apply options to a clone and detect config drift.
	// CreateBackend is intentionally NOT called here so we do not hold the mutex
	// for up to 30 s while a remote endpoint is contacted.
	err := func() error {
		p.mutex.Lock()
		defer p.mutex.Unlock()

		// Capture the pre-update state needed for correct lifecycle-hook decisions.
		wasEnabled = p.storedInfo.GetEnabled()
		priorBackend = p.backend

		// Try updates on a copy of the provider.
		modifiedProvider = cloneWithoutMutex(p)
		if err := applyOptions(modifiedProvider, options...); err != nil {
			return err
		}

		// Detect backend config drift: if the stored config was changed by an option
		// (e.g. WithConfig) and the existing backend was built from the old config,
		// we must rebuild the backend so it reflects the new configuration.
		if modifiedProvider.backend != nil && modifiedProvider.backendFactory != nil &&
			!maps.Equal(modifiedProvider.storedInfo.GetConfig(), modifiedProvider.backend.Config()) {
			// Remember the stale backend so we can call OnDisable after releasing
			// the mutex (using the captured pre-update wasEnabled flag from Comment 3).
			staleBackend = modifiedProvider.backend
			modifiedProvider.backend = nil
			needsBackendRebuild = true
			// Do not copyWithoutMutex yet; we will do it after the rebuild below.
			return nil
		}

		// No drift: commit the updated clone to the real provider now.
		copyWithoutMutex(p, modifiedProvider)
		return nil
	}()
	if err != nil {
		return err
	}

	// Step 2: if config drift was detected, rebuild the backend outside the lock
	// so that no mutex is held during the potentially slow CreateBackend call.
	if needsBackendRebuild {
		ctx, cancel := context.WithTimeout(context.Background(), asyncBackendCreationTimeout)
		defer cancel()
		newBackend, err := modifiedProvider.backendFactory.CreateBackend(
			ctx,
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

		// Step 3: commit the rebuilt clone under the lock.
		p.mutex.Lock()
		copyWithoutMutex(p, modifiedProvider)
		p.mutex.Unlock()
	}

	// Step 4: fire lifecycle hooks outside the mutex so that any provider accessor
	// methods called from within the hooks can safely acquire the read lock.
	//
	// OnDisable for the stale backend: use wasEnabled (pre-update state, Comment 3)
	// so we correctly clean up regardless of whether the provider is now enabled.
	if staleBackend != nil && wasEnabled {
		staleBackend.OnDisable(p)
	}
	// OnEnable for a brand-new backend: only when ApplyOptions itself introduced
	// the backend (priorBackend was nil) and no subsequent updatedNoLock call will
	// fire OnEnable — e.g. the WithBackendFromFactory late-registration path.
	// Read the new backend and enabled state from modifiedProvider (the committed
	// clone) to avoid an unlocked read of p's fields.
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
