package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/jwt"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	factory        tokens.IssuerFactory
	tokenValidator tokens.Validator

	initOnce sync.Once
)

const (
	privateKeyDir     = "/run/secrets/stackrox.io/jwt"
	privateKeyPath    = privateKeyDir + "/jwt-key.der"
	privateKeyPathPEM = privateKeyDir + "/jwt-key.pem"
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
	if _, err := os.Stat(privateKeyPath); err == nil {
		return os.ReadFile(privateKeyPath)
	} else if _, err := os.Stat(privateKeyPathPEM); err == nil {
		// Second attempt: Try reading PEM version and convert.
		return getBytesFromPem(privateKeyPathPEM)
	} else {
		return nil, errors.Wrap(err, "could not load private key")
	}
}

func create() (tokens.IssuerFactory, tokens.Validator, error) {
	// Load initial key so that we would immediately fail in case
	// it's not present.
	initialPrivateKey, err := loadPrivateKey(privateKeyDir)
	if err != nil {
		return nil, nil, errors.Wrap(err, "loading initial value of JWT private key")
	}
	privateKeyStore := jwt.NewSinglePrivateKeyStore(initialPrivateKey, keyID)
	publicKeyStore := jwt.NewDerivedPublicKeyStore(privateKeyStore, keyID)
	jwt.WatchKeyDir(privateKeyDir, loadPrivateKey, func(key *rsa.PrivateKey) {
		privateKeyStore.UpdateKey(keyID, key)
	})

	issuerFactory, validator := tokens.CreateIssuerFactoryAndValidator(issuerID, privateKeyStore, publicKeyStore, keyID)
	return issuerFactory, validator, nil
}

// We pass parameter here to satisfy WatchKeyDir interface.
func loadPrivateKey(_ string) (*rsa.PrivateKey, error) {
	privateKeyBytes, err := GetPrivateKeyBytes()
	if err != nil {
		return nil, errors.Wrap(err, "loading private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
	if err != nil {
		return nil, errors.Wrap(err, "parsing private key")
	}
	return privateKey, nil
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
