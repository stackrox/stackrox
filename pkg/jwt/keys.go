package jwt

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"gopkg.in/square/go-jose.v2"
)

const (
	httpTimeout = 10 * time.Second
)

// KeyGetter is the interface that all providers of JSON Web Keys should implement.
type KeyGetter interface {
	// The key should be a type that go-jose understands. Valid types include:
	// ed25519.PublicKey
	// *rsa.PublicKey
	// *ecdsa.PublicKey
	// []byte
	// jose.JSONWebKey
	// *jose.JSONWebKey
	Key(id string) interface{}
}

// A JWKSGetter gets trusted keys from a JSON Web Key Set (JWKS) URL.
type JWKSGetter struct {
	url       string
	fetchOnce sync.Once
	known     map[string]*jose.JSONWebKey
}

// NewJWKSGetter creates a new KeyGetter that gets trusted keys from a JSON Web Key Set (JWKS) URL.
func NewJWKSGetter(url string) *JWKSGetter {
	return &JWKSGetter{
		url:   url,
		known: make(map[string]*jose.JSONWebKey),
	}
}

// Key returns the key with the given ID, if it is in the set.
func (j *JWKSGetter) Key(id string) interface{} {
	j.fetchOnce.Do(j.fetch)
	key := j.known[id]
	if key == nil {
		return nil // note this is NOT equivalent to returning `key`, which is (*JSONWebKey)(nil), not the nil interface{}.
	}
	return key
}

func (j *JWKSGetter) fetch() {
	cli := http.Client{
		Timeout: httpTimeout,
	}
	resp, err := cli.Get(j.url)
	if err != nil {
		// TODO(cg)
		log.Warnf("Couldn't get JWKS URL '%s': %s", j.url, err)
		return
	}
	defer utils.IgnoreError(resp.Body.Close)
	dec := json.NewDecoder(resp.Body)
	var jwks jose.JSONWebKeySet
	if err := dec.Decode(&jwks); err != nil {
		// TODO(cg)
		log.Warnf("Couldn't decode JWKS response: %s", err)
		return
	}
	for _, jwk := range jwks.Keys {
		jwkCopy := jwk
		j.known[jwkCopy.KeyID] = &jwkCopy
	}
}

type SingleKeyStore struct {
	keyID string
	key   interface{}
	mutex sync.RWMutex
}

func (s *SingleKeyStore) Key(id string) interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if id == s.keyID {
		return s.key
	}
	log.Error("not found")
	return nil
}

func (s *SingleKeyStore) UpdateKey(newVal interface{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.key = newVal
	log.Infof("Value is updated: %v", s.key)
}

// NewSingleKeyStore returns a KeyGetter that allows obtaining a single key with a defined id.
func NewSingleKeyStore(key interface{}, keyID string) *SingleKeyStore {
	return &SingleKeyStore{
		keyID: keyID,
		key:   key,
	}
}
