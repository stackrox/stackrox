package service

import (
	imagev2DS "github.com/stackrox/rox/central/imagev2/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	srv Service
)

func initialize() {
	srv = New(imagev2DS.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return srv
}
