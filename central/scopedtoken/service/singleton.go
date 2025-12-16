package service

import (
	"github.com/stackrox/rox/central/apitoken/backend"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

func initialize() {
	svc = New(backend.Singleton())
}

// Singleton returns the scoped token service singleton.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
