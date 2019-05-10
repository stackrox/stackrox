package service

import (
	"github.com/stackrox/rox/central/apitoken/backend"
	rolestore "github.com/stackrox/rox/central/role/store"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	svc = New(backend.Singleton(), rolestore.Singleton())
}

// Singleton returns the API token singleton.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
