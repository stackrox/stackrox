package service

import (
	"github.com/stackrox/stackrox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/stackrox/central/notifier/processor"
	baselineDatastore "github.com/stackrox/stackrox/central/processbaseline/datastore"
	"github.com/stackrox/stackrox/central/sensor/service/connection"
	"github.com/stackrox/stackrox/pkg/sync"
)

var (
	once         sync.Once
	soleInstance Service
)

func initialize() {
	soleInstance = New(datastore.Singleton(), baselineDatastore.Singleton(), notifierProcessor.Singleton(), connection.ManagerSingleton())
}

// Singleton returns the sole instance of the gRPC Server Service for handling CRUD use cases for Alert objects.
func Singleton() Service {
	once.Do(initialize)
	return soleInstance
}
