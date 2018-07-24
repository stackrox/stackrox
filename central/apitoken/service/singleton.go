package service

import (
	"sync"

	"bitbucket.org/stack-rox/apollo/central/apitoken/signer"
	"bitbucket.org/stack-rox/apollo/central/role/store"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	svc = New(signer.Singleton(), store.Singleton())
}

// Singleton returns the API token singleton.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
