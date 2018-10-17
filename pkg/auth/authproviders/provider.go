package authproviders

import (
	"errors"
	"sync"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/auth/tokens"
)

type authProvider struct {
	backend    AuthProviderBackend
	baseInfo   v1.AuthProvider
	roleMapper permissions.RoleMapper

	registry *storeBackedRegistry
	issuer   tokens.Issuer

	mutex sync.RWMutex
}

func (p *authProvider) Validate(claims *tokens.Claims) error {
	// Signature validation/expiry checks/check for enabledness is already done by the JWT layer.
	// TODO: allow the backend to do validation?
	return nil
}

func (p *authProvider) RoleMapper() permissions.RoleMapper {
	return p.roleMapper
}

func (p *authProvider) RecordSuccess() error {
	if !p.Enabled() {
		return errors.New("cannot record success for disabled auth provider")
	}
	if err := p.registry.recordSuccess(p.ID()); err != nil {
		return err
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.baseInfo.Validated = true
	return nil
}

func (p *authProvider) Backend() AuthProviderBackend {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.backend
}

func (p *authProvider) ID() string {
	return p.baseInfo.GetId()
}

func (p *authProvider) Enabled() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.backend != nil && p.baseInfo.Enabled
}

func (p *authProvider) Validated() bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.baseInfo.Validated
}

func (p *authProvider) Type() string {
	return p.baseInfo.Type
}

func (p *authProvider) Name() string {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	return p.baseInfo.Name
}

func (p *authProvider) setBackend(backend AuthProviderBackend, effectiveConfig map[string]string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.backend = backend
	p.baseInfo.Config = effectiveConfig
}

func (p *authProvider) update(name *string, enabled *bool) (bool, v1.AuthProvider, string) {
	modified := false
	var oldName string

	p.mutex.Lock()
	defer p.mutex.Unlock()
	if name != nil && *name != p.baseInfo.Name {
		oldName = p.baseInfo.Name
		p.baseInfo.Name = *name
		modified = true
	}
	if enabled != nil && *enabled != p.baseInfo.Enabled {
		p.baseInfo.Enabled = *enabled
		modified = true
	}
	return modified, p.baseInfo, oldName
}

func (p *authProvider) AsV1(clientState string) *v1.AuthProvider {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	result := p.baseInfo
	if p.backend == nil {
		result.Enabled = false
	}
	if result.GetEnabled() {
		result.LoginUrl = p.backend.LoginURL(clientState)
	}

	return &result
}
