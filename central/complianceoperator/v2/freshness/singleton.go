package freshness

import (
	"time"

	checkResultDS "github.com/stackrox/rox/central/complianceoperator/v2/checkresults/datastore"
	scanConfigDS "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
)

// minSweepInterval is the floor enforced in production so the sweeper never runs more than once a
// minute, regardless of how ROX_COMPLIANCE_FRESHNESS_SWEEP_INTERVAL is configured.
const minSweepInterval = time.Minute

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
			clampSweepInterval(env.ComplianceFreshnessSweepInterval.DurationSetting()),
		)
	})
	return singleton
}

// clampSweepInterval enforces minSweepInterval as a floor so a misconfigured, overly aggressive
// interval cannot make the sweeper run more often than once a minute in production.
func clampSweepInterval(d time.Duration) time.Duration {
	if d < minSweepInterval {
		log.Warnf("compliance freshness sweep interval %s is below the minimum %s; clamping to %s", d, minSweepInterval, minSweepInterval)
		return minSweepInterval
	}
	return d
}
