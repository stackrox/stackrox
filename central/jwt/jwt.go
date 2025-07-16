package jwt

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/auth/m2m"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
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
	issuerID          = "https://stackrox.io/jwt"

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
	issuerID     string
}

// Validate implements tokens.Validator.
func (v *m2mValidator) Validate(ctx context.Context, token string) (*tokens.TokenInfo, error) {
	if !v.HasExchangersConfigured() {
		return v.roxValidator.Validate(ctx, token)
	}
	iss, err := m2m.IssuerFromRawIDToken(token)
	if err != nil {
		return nil, err
	}
	if iss == v.issuerID {
		return v.roxValidator.Validate(ctx, token)
	}
	exchanger, found := v.GetTokenExchanger(iss)
	if !found {
		return nil, errox.NoCredentials.CausedBy("no exchanger found for issuer " + iss)
	}
	newToken, err := exchanger.ExchangeToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return v.roxValidator.Validate(ctx, newToken)
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

	factory, validator, err := tokens.CreateIssuerFactoryAndValidator(issuerID, privateKey, keyID)
	if err != nil {
		return nil, nil, errors.Wrap(err, "creating factory and validator")
	}
	validator = &m2mValidator{
		m2m.TokenExchangerSetSingleton(roleDataStore.Singleton(), IssuerFactorySingleton()),
		validator,
		issuerID,
	}
	return factory, validator, err
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
