package service

import (
	datastore "github.com/stackrox/rox/central/administration/usage/datastore/securedunits"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	svc  Service
	once sync.Once
)

// Singleton returns the administration usage service singleton.
func Singleton() Service {
	once.Do(func() {
		svc = New(datastore.Singleton())
	})
	return svc
}
