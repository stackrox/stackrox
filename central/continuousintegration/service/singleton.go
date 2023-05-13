package service

import (
	"github.com/stackrox/rox/central/continuousintegration/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Service
)

// Singleton returns the continuous integration service singleton.
func Singleton() Service {
	once.Do(func() {
		instance = New(datastore.Singleton())
	})
	return instance
}
