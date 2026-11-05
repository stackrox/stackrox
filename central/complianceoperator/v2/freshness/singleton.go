package freshness

import (
	checkResultDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once      sync.Once
	singleton Sweeper
)

// Singleton returns the compliance data freshness sweeper singleton.
func Singleton() Sweeper {
	once.Do(func() {
		singleton = New(
			scanConfigDS.Singleton(),
			checkResultDS.Singleton(),
			env.ComplianceFreshnessSweepInterval.DurationSetting(),
		)
	})
	return singleton
}
