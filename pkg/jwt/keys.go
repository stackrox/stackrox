package jwt

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

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
	Key(id string) (key interface{}, found bool)
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
func (j *JWKSGetter) Key(id string) (interface{}, bool) {
	j.fetchOnce.Do(j.fetch)
	key, found := j.known[id]
	return key, found
}

func (j *JWKSGetter) fetch() {
	cli := http.Client{
		Timeout: httpTimeout,
	}
	resp, err := cli.Get(j.url)
	if err != nil {
		// TODO(cg)
		logger.Warnf("Couldn't get JWKS URL '%s': %s", j.url, err)
		return
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	var jwks jose.JSONWebKeySet
	if err := dec.Decode(&jwks); err != nil {
		// TODO(cg)
		logger.Warnf("Couldn't decode JWKS response: %s", err)
		return
	}
	for _, jwk := range jwks.Keys {
		j.known[jwk.KeyID] = &jwk
	}
}
