package service

import (
	imageDS "github.com/stackrox/rox/central/image/datastore"
	"github.com/stackrox/rox/central/views/cveexport"
	"github.com/stackrox/rox/central/views/vulnfinding"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	srv Service
)

func initialize() {
	srv = New(imageDS.Singleton(), cveexport.Singleton(), vulnfinding.Singleton())
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return srv
}
