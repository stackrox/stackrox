package service

import (
	"github.com/stackrox/rox/central/alert/datastore"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	baselineDatastore "github.com/stackrox/rox/central/processbaseline/datastore"
	"github.com/stackrox/rox/central/sensor/service/connection"
	"github.com/stackrox/rox/pkg/sync"
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
