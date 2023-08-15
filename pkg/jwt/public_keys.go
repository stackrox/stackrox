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

// PublicKeyGetter is the interface that all providers of JSON Web Keys should implement.
type PublicKeyGetter interface {
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

// NewJWKSGetter creates a new PublicKeyGetter that gets trusted keys from a JSON Web Key Set (JWKS) URL.
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

// derivedPublicKeyStore provides public key by looking up into underlying private key store.
type derivedPublicKeyStore struct {
	keyID           string
	privateKeyStore PrivateKeyGetter
}

// NewDerivedPublicKeyStore returns a PublicKeyGetter that allows obtaining a single, derived from private key with a defined id.
func NewDerivedPublicKeyStore(privateKeyStore PrivateKeyGetter, keyID string) PublicKeyGetter {
	return &derivedPublicKeyStore{
		keyID:           keyID,
		privateKeyStore: privateKeyStore,
	}
}

// Key returns public key with matching id.
func (d *derivedPublicKeyStore) Key(id string) interface{} {
	if id != d.keyID {
		return nil
	}
	privateKey := d.privateKeyStore.Key(d.keyID)
	if privateKey != nil {
		return privateKey.Public()
	}
	return nil
}
