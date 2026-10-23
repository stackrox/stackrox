package freshness

import (
	checkResultDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	once     sync.Once
	instance Sweeper
)

// Singleton returns the singleton instance of the compliance freshness Sweeper.
func Singleton() Sweeper {
	once.Do(func() {
		instance = New(scanConfigDS.Singleton(), checkResultDS.Singleton(), env.ComplianceFreshnessSweepInterval.DurationSetting())
	})
	return instance
}
