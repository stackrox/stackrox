package service

import (
	"github.com/stackrox/stackrox/central/reprocessor"
	"github.com/stackrox/stackrox/central/signatureintegration/datastore"
	"github.com/stackrox/stackrox/pkg/sync"
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
