package service

import (
	"github.com/stackrox/rox/central/sac/datastore"
	"github.com/stackrox/rox/pkg/sync"
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
