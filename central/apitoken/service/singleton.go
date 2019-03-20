package service

import (
	"github.com/stackrox/rox/pkg/sync"

	"github.com/stackrox/rox/central/apitoken"
	rolestore "github.com/stackrox/rox/central/role/store"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	svc = New(apitoken.BackendSingleton(), rolestore.Singleton())
}

// Singleton returns the API token singleton.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
