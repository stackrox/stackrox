package service

import (
	"github.com/stackrox/rox/central/signature/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Service
)

func initialize() {
	instance = New(datastore.Singleton())
}

// Singleton returns the signature service singleton.
func Singleton() Service {
	once.Do(initialize)
	return instance
}
