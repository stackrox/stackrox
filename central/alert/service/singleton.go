package service

import (
	"github.com/stackrox/rox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	whitelistDatastore "github.com/stackrox/rox/central/processwhitelist/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once         sync.Once
	soleInstance Service
)

func initialize() {
	soleInstance = New(datastore.Singleton(), whitelistDatastore.Singleton(), notifierProcessor.Singleton())
}

// Singleton returns the sole instance of the gRPC Server Service for handling CRUD use cases for Alert objects.
func Singleton() Service {
	once.Do(initialize)
	return soleInstance
}
