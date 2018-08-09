package service

import (
	"sync"

	"github.com/stackrox/rox/central/apitoken/cachedstore"
	"github.com/stackrox/rox/central/apitoken/parser"
	"github.com/stackrox/rox/central/apitoken/signer"
	rolestore "github.com/stackrox/rox/central/role/store"
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
