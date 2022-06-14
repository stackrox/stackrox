package service

import (
	"github.com/stackrox/stackrox/central/sac/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once sync.Once
	as   Service
)

// Singleton for the sac service.  This is use to configure auth plugins.
func Singleton() Service {
	once.Do(func() {
		as = New(datastore.Singleton())
	})
	return as
}
