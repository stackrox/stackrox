package authproviders

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

// If you add new data fields to this class, make sure you make commensurate modifications
// to the cloneWithoutMutex and copyWithoutMutex functions below.
type providerImpl struct {
	mutex sync.RWMutex

	storedInfo storage.AuthProvider
	backend    Backend
	roleMapper permissions.RoleMapper
	issuer     tokens.Issuer
	onSuccess  ProviderOption

	doNotStore bool
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

	return p.storedInfo.Type
}

func (p *providerImpl) Name() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.storedInfo.Name
}

func (p *providerImpl) Enabled() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.backend != nil && p.storedInfo.Enabled
}

func (p *providerImpl) Validated() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return p.storedInfo.Validated
}

func (p *providerImpl) StorageView() *storage.AuthProvider {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	result := p.storedInfo
	if p.backend == nil {
		result.Enabled = false
	}
	return &result
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

// Modifier functions.
//////////////////////

func (p *providerImpl) OnSuccess() error {
	return p.applyOptions(p.onSuccess)
}

func (p *providerImpl) Validate(claims *tokens.Claims) error {
	// Signature validation/expiry checks/check for enabledness is already done by the JWT layer.
	// TODO: allow the backend to do validation?
	return nil
}

// We must lock the provider when applying options to it.
func (p *providerImpl) applyOptions(options ...ProviderOption) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Try updates on a copy of the provider
	modifiedProvider := cloneWithoutMutex(p)
	if err := applyOptions(modifiedProvider, options...); err != nil {
		return err
	}

	// If updates succeed, apply them.
	copyWithoutMutex(p, modifiedProvider)
	return nil
}

// Does a deep copy of the proto field 'storedInfo' so that it can support nested message fields.
func cloneWithoutMutex(pr *providerImpl) *providerImpl {
	return &providerImpl{
		storedInfo: *proto.Clone(&pr.storedInfo).(*storage.AuthProvider),
		backend:    pr.backend,
		roleMapper: pr.roleMapper,
		issuer:     pr.issuer,
		onSuccess:  pr.onSuccess,
	}
}

// No need to do a deep copy of the 'storedInfo' field here since the 'from' input was created with a deep copy.
func copyWithoutMutex(to *providerImpl, from *providerImpl) {
	to.storedInfo = from.storedInfo
	to.backend = from.backend
	to.roleMapper = from.roleMapper
	to.issuer = from.issuer
	to.onSuccess = from.onSuccess
}
