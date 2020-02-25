package basic

import (
	"sync/atomic"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/htpasswd"
)

// Manager manages basic auth user identities.
type Manager struct {
	hashFilePtr unsafe.Pointer
	userRole    *storage.Role
}

func (m *Manager) hashFile() *htpasswd.HashFile {
	return (*htpasswd.HashFile)(atomic.LoadPointer(&m.hashFilePtr))
}

// SetHashFile sets the hash file to be used for basic auth.
func (m *Manager) SetHashFile(hashFile *htpasswd.HashFile) {
	atomic.StorePointer(&m.hashFilePtr, unsafe.Pointer(hashFile))
}

// IdentityForCreds returns an identity for the given credentials.
func (m *Manager) IdentityForCreds(username, password string, authProvider authproviders.Provider) (Identity, error) {
	if !m.hashFile().Check(username, password) {
		return nil, errors.New("invalid username and/or password")
	}

	return identity{
		username:     username,
		role:         m.userRole,
		authProvider: authProvider,
	}, nil
}

// NewManager creates a new manager for basic authentication.
func NewManager(hashFile *htpasswd.HashFile, userRole *storage.Role) *Manager {
	return &Manager{
		hashFilePtr: unsafe.Pointer(hashFile),
		userRole:    userRole,
	}
}
