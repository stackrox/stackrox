package idputil

import (
	"crypto/rand"
	"time"

	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timeutil"
)

const (
	nonceByteLen = 20
	stateTTL     = 5 * time.Minute
)

var (
	errNonceUnknown = errox.InvalidArgs.New("unknown or expired state nonce")

	globalStore     *StateStore
	globalStoreOnce sync.Once
)

type stateData struct {
	providerID  string
	clientState string
	expiresAt   time.Time
}

// StateStore maps opaque nonces to auth provider state data. Nonces are
// single-use when redeemed via Redeem, and expire after the configured TTL.
type StateStore struct {
	entries    map[string]stateData
	mutex      sync.Mutex
	generator  cryptoutils.NonceGenerator
	ttl        time.Duration
	nextExpiry time.Time
}

// NewStateStore creates a new state store with the given TTL.
func NewStateStore(ttl time.Duration) *StateStore {
	return &StateStore{
		entries:    make(map[string]stateData),
		generator:  cryptoutils.NewNonceGenerator(nonceByteLen, rand.Reader),
		ttl:        ttl,
		nextExpiry: timeutil.Max,
	}
}

// GlobalStateStore returns the process-wide state store.
func GlobalStateStore() *StateStore {
	globalStoreOnce.Do(func() {
		globalStore = NewStateStore(stateTTL)
	})
	return globalStore
}

// Issue generates a nonce and stores the associated state data.
func (s *StateStore) Issue(providerID, clientState string) (string, error) {
	nonce, err := s.generator.Nonce()
	if err != nil {
		return "", err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	expiry := time.Now().Add(s.ttl)
	s.entries[nonce] = stateData{
		providerID:  providerID,
		clientState: clientState,
		expiresAt:   expiry,
	}
	if s.nextExpiry.After(expiry) {
		s.nextExpiry = expiry
	} else {
		s.cleanup()
	}
	return nonce, nil
}

// Redeem returns the state data for the nonce and deletes it (single-use).
func (s *StateStore) Redeem(nonce string) (providerID, clientState string, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, ok := s.entries[nonce]
	if ok {
		delete(s.entries, nonce)
		if data.expiresAt.Before(time.Now()) {
			ok = false
		}
	}
	s.cleanup()
	if !ok {
		return "", "", errNonceUnknown
	}
	return data.providerID, data.clientState, nil
}

// Lookup returns the state data for the nonce without consuming it.
func (s *StateStore) Lookup(nonce string) (providerID, clientState string, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	data, ok := s.entries[nonce]
	if !ok || data.expiresAt.Before(time.Now()) {
		s.cleanup()
		return "", "", errNonceUnknown
	}
	return data.providerID, data.clientState, nil
}

func (s *StateStore) cleanup() {
	now := time.Now()
	if !now.After(s.nextExpiry) {
		return
	}
	nextExpiry := timeutil.Max
	for nonce, data := range s.entries {
		if data.expiresAt.Before(now) {
			delete(s.entries, nonce)
		} else if data.expiresAt.Before(nextExpiry) {
			nextExpiry = data.expiresAt
		}
	}
	s.nextExpiry = nextExpiry
}

// IssueStateNonce issues a nonce for the given provider ID and client state.
func IssueStateNonce(providerID, clientState string) (string, error) {
	return GlobalStateStore().Issue(providerID, clientState)
}

// RedeemStateNonce redeems a nonce and returns the stored state (single-use).
func RedeemStateNonce(nonce string) (providerID, clientState string, err error) {
	return GlobalStateStore().Redeem(nonce)
}

// LookupStateNonce looks up a nonce without consuming it.
func LookupStateNonce(nonce string) (providerID, clientState string, err error) {
	return GlobalStateStore().Lookup(nonce)
}
