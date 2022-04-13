package service

import (
	"github.com/stackrox/stackrox/central/apitoken/backend"
	roleDS "github.com/stackrox/stackrox/central/role/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	svc = New(backend.Singleton(), roleDS.Singleton())
}

// Singleton returns the API token singleton.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
