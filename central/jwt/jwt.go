package jwt

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/m2m"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/expiringcache"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	factory        tokens.IssuerFactory
	tokenValidator tokens.Validator

	initOnce sync.Once
)

const (
	privateKeyPath    = "/run/secrets/stackrox.io/jwt/jwt-key.der"
	privateKeyPathPEM = "/run/secrets/stackrox.io/jwt/jwt-key.pem"
	roxIssuer         = "https://stackrox.io/jwt"

	keyID = "jwtk0"
)

func getBytesFromPem(path string) ([]byte, error) {
	bytesPemEncoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoded, _ := pem.Decode(bytesPemEncoded)
	if decoded == nil {
		return nil, errors.Errorf("invalid PEM in %s", path)
	}
	return decoded.Bytes, nil
}

// GetPrivateKeyBytes returns the contents of the file containing the private key.
func GetPrivateKeyBytes() ([]byte, error) {
	_, err := os.Stat(privateKeyPath)
	if err == nil {
		return os.ReadFile(privateKeyPath)
	}
	_, err = os.Stat(privateKeyPathPEM)
	if err != nil {
		return nil, errors.Wrap(err, "could not load private key")
	}
	// Second attempt: Try reading PEM version and convert.
	return getBytesFromPem(privateKeyPathPEM)
}

type m2mValidator struct {
	m2m.TokenExchangerSet
	roxValidator tokens.Validator
	// A token cache per issuer with separately configured TTL.
	exchangedTokensCache map[string]expiringcache.Cache[string, string]
	cacheMux             sync.Mutex
}

// Validate the token: if this is not a stackrox.io token, exchange it first,
// according to the M2M configuration, and then validate using roxValidator.
func (v *m2mValidator) Validate(ctx context.Context, token string) (*tokens.TokenInfo, error) {
	// Short-circuit here in case there are no M2M configuration available for
	// the implicit token exchange.
	if !v.HasExchangersConfigured() {
		return v.roxValidator.Validate(ctx, token)
	}
	iss, err := m2m.IssuerFromRawIDToken(token)
	if err != nil {
		return nil, err
	}
	// If this is a stackrox.io token, let's just validate it.
	if iss == roxIssuer {
		return v.roxValidator.Validate(ctx, token)
	}
	// Otherwise, let's exchange the token according to an M2M configuration for
	// this issuer, if available.
	newToken, err := v.exchange(ctx, iss, token)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to exchange an ID token for issuer %s", iss)
	}
	return v.roxValidator.Validate(ctx, newToken)
}

func (v *m2mValidator) exchange(ctx context.Context, iss string, token string) (string, error) {
	exchanger, found := v.GetTokenExchanger(iss)
	if !found {
		return "", errox.NoCredentials.CausedBy("no exchanger found")
	}

	v.cacheMux.Lock()
	defer v.cacheMux.Unlock()

	cache, found := v.exchangedTokensCache[iss]
	if !found {
		if tokenTTL, err := time.ParseDuration(exchanger.Config().GetTokenExpirationDuration()); err == nil {
			// TTL for the cached token should be less than the token TTL so that
			// the cache doen't return expired tokens. Let's not cache tokens with
			// TTL less than a 1 minute margin.
			const margin = time.Minute
			if tokenTTL > margin {
				cache = expiringcache.NewExpiringCache[string, string](tokenTTL - margin)
			}
		}
		v.exchangedTokensCache[iss] = cache
	}
	if cache == nil {
		// No cache, just exchange:
		return exchanger.ExchangeToken(ctx, token)
	}
	newToken, found := cache.Get(token)
	if !found {
		var err error
		// The exchanger will pass the provided token to the according
		// coreos/go-oidc provider for verification (expiration, signature, etc.).
		newToken, err = exchanger.ExchangeToken(ctx, token)
		if err != nil {
			return "", err
		}
		cache.Add(token, newToken)
	}
	return newToken, nil
}

func create() (tokens.IssuerFactory, tokens.Validator, error) {
	privateKeyBytes, err := GetPrivateKeyBytes()
	if err != nil {
		return nil, nil, errors.Wrap(err, "loading private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
	if err != nil {
		return nil, nil, errors.Wrap(err, "parsing private key")
	}

	factory, validator, err := tokens.CreateIssuerFactoryAndValidator(roxIssuer, privateKey, keyID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating factory and validator")
	}
	// Decorate the stackrox.io token validator with M2M validator in case the
	// provided token is not issued by stackrox.io and requires an implicit
	// exchange.
	exchangerSet := m2m.TokenExchangerSetSingleton(roleDataStore.Singleton(), factory)
	return factory, &m2mValidator{exchangerSet, validator,
		make(map[string]expiringcache.Cache[string, string]), sync.Mutex{},
	}, err
}

func initialize() {
	var err error
	factory, tokenValidator, err = create()
	if err != nil {
		log.Panicf("Could not instantiate JWT factory: %v", err)
	}
}

// singleton returns the singleton issuer factory & validator.
func singleton() (tokens.IssuerFactory, tokens.Validator) {
	initOnce.Do(initialize)
	return factory, tokenValidator
}

// IssuerFactorySingleton retrieves the issuer factory singleton instance.
func IssuerFactorySingleton() tokens.IssuerFactory {
	factory, _ := singleton()
	return factory
}

// ValidatorSingleton retrieves the validator singleton instance.
func ValidatorSingleton() tokens.Validator {
	_, validator := singleton()
	return validator
}
