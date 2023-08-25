package service

import (
	"github.com/stackrox/rox/central/reprocessor"
	"github.com/stackrox/rox/central/signatureintegration/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Service
)

// Singleton returns the signature integration service singleton.
func Singleton() Service {
	once.Do(func() {
		instance = New(datastore.Singleton(), reprocessor.Singleton())
	})
	return instance
}
