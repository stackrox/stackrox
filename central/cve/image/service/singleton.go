package service

import (
	imageCVEV2DS "github.com/stackrox/rox/central/cve/image/v2/datastore"
	imageComponentV2DS "github.com/stackrox/rox/central/imagecomponent/v2/datastore"
	"github.com/stackrox/rox/central/views/imagecve"
	vulnReqDataStore "github.com/stackrox/rox/central/vulnmgmt/vulnerabilityrequest/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once
	as   Service
)

func initialize() {
	as = New(
		imagecve.Singleton(),
		imageCVEV2DS.Singleton(),
		imageComponentV2DS.Singleton(),
		vulnReqDataStore.Singleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return as
}
