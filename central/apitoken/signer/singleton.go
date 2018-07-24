package signer

import (
	"fmt"
	"io/ioutil"
	"sync"
)

var (
	signer Signer
	once   sync.Once
)

const (
	privateKeyPath = "/run/secrets/stackrox.io/jwt/jwt-key.der"
)

func initialize() {
	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		panic(fmt.Errorf("couldn't load private key for API token signer: %s", err))
	}
	signer, err = NewFromBytes(privateKeyBytes)
	if err != nil {
		panic(fmt.Errorf("couldn't initialize API key signer: %s", err))
	}
}

// Singleton returns the signer's singleton.
func Singleton() Signer {
	once.Do(initialize)
	return signer
}
