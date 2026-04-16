package service

import (
	"github.com/stackrox/rox/central/views/vmcve"
	componentDS "github.com/stackrox/rox/central/virtualmachine/component/v2/datastore"
	cveDS "github.com/stackrox/rox/central/virtualmachine/cve/v2/datastore"
	scanDS "github.com/stackrox/rox/central/virtualmachine/scan/v2/datastore"
	vmDS "github.com/stackrox/rox/central/virtualmachine/v2/datastore"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once sync.Once

	svc Service
)

func initialize() {
	svc = New(
		vmDS.Singleton(),
		cveDS.Singleton(),
		componentDS.Singleton(),
		scanDS.Singleton(),
		vmcve.Singleton(),
	)
}

// Singleton provides the instance of the Service interface to register.
func Singleton() Service {
	once.Do(initialize)
	return svc
}
