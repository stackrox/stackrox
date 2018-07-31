package service

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/apitoken/cachedstore"
	"bitbucket.org/stack-rox/apollo/central/apitoken/parser"
	"bitbucket.org/stack-rox/apollo/central/apitoken/signer"
	rolestore "bitbucket.org/stack-rox/apollo/central/role/store"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	svc = New(signer.Singleton(), parser.Singleton(), rolestore.Singleton(), cachedstore.Singleton())
}

// Singleton returns the API token singleton.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
