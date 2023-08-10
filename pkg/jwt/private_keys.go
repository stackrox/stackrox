package jwt

import (
	"crypto"

	"github.com/stackrox/rox/pkg/sync"
)

// PrivateKeyGetter allows to obtain JWT private keys.
// Note: the reason we use crypto.Signer here is because crypto.PrivateKey
// is empty interface that does not implement Public() method.
type PrivateKeyGetter interface {
	Key(keyID string) crypto.Signer
}

// PrivateKeySetter allows to set JWT private keys.
// Note: the reason we use crypto.Signer here is because crypto.PrivateKey
// is empty interface that does not implement Public() method.
type PrivateKeySetter interface {
	UpdateKey(keyID string, key crypto.Signer)
}

type PrivateKeyStore interface {
	PrivateKeyGetter
	PrivateKeySetter
}

type singlePrivateKeyStore struct {
	keyID string
	key   crypto.Signer
	mutex sync.RWMutex
}

func (s *singlePrivateKeyStore) Key(keyID string) crypto.Signer {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if keyID == s.keyID {
		return s.key
	}
	return nil
}

func (s *singlePrivateKeyStore) UpdateKey(keyID string, newVal crypto.Signer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if keyID == s.keyID {
		s.key = newVal
	}
}

// NewSinglePrivateKeyStore returns PrivateKeyGetter that allows obtaining a single key with a defined id.
func NewSinglePrivateKeyStore(key crypto.Signer, keyID string) PrivateKeyStore {
	return &singlePrivateKeyStore{
		keyID: keyID,
		key:   key,
	}
}
