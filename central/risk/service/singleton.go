package service

import (
	"github.com/stackrox/rox/central/risk/manager"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	svc  Service
)

// Singleton returns the singleton risk service instance.
func Singleton() Service {
	once.Do(func() {
		svc = New(manager.Singleton())
	})
	return svc
}
