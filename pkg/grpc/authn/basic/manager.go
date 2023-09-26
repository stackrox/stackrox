package basic

import (
	"context"
	"sync/atomic"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/htpasswd"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/errox"
)

// Manager manages basic auth user identities.
type Manager struct {
	hashFilePtr unsafe.Pointer
	mapper      permissions.RoleMapper
}

func (m *Manager) hashFile() *htpasswd.HashFile {
	return (*htpasswd.HashFile)(atomic.LoadPointer(&m.hashFilePtr))
}

// SetHashFile sets the hash file to be used for basic auth.
func (m *Manager) SetHashFile(hashFile *htpasswd.HashFile) {
	//#nosec G103
	atomic.StorePointer(&m.hashFilePtr, unsafe.Pointer(hashFile))
}

// IdentityForCreds returns an identity for the given credentials.
func (m *Manager) IdentityForCreds(ctx context.Context, username, password string, authProvider authproviders.Provider) (Identity, error) {
	if !m.hashFile().Check(username, password) {
		return nil, errox.NotAuthorized.CausedBy("invalid username and/or password")
	}

	resolvedRoles, err := m.mapper.FromUserDescriptor(ctx, &permissions.UserDescriptor{
		UserID:     username,
		Attributes: map[string][]string{},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to load roles for user %q", username)
	}

	return identity{
		username:      username,
		resolvedRoles: resolvedRoles,
		authProvider:  authProvider,
	}, nil
}

// NewManager creates a new manager for basic authentication.
func NewManager(hashFile *htpasswd.HashFile, roleMapper permissions.RoleMapper) *Manager {
	return &Manager{
		//#nosec G103
		hashFilePtr: unsafe.Pointer(hashFile),
		mapper:      roleMapper,
	}
}
