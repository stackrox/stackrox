package jwt

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"sync"

	"github.com/stackrox/rox/pkg/auth/tokens"
)

var (
	factory        tokens.IssuerFactory
	tokenValidator tokens.Validator

	initOnce sync.Once
)

const (
	privateKeyPath = "/run/secrets/stackrox.io/jwt/jwt-key.der"
	issuerID       = "https://stackrox.io/jwt"

	keyID = "jwtk0"
)

func create() (tokens.IssuerFactory, tokens.Validator, error) {
	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("loading private key: %v", err)
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing private key: %v", err)
	}

	return tokens.CreateIssuerFactoryAndValidator(issuerID, privateKey, keyID)
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
