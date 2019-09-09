package upgradecontroller

import (
	"time"
)

var (
	defaultTimeoutProvider = timeoutProvider{
		upgraderStartGracePeriod:    30 * time.Second,
		stuckInSameStateTimeout:     5 * time.Minute,
		stateReconcilerPollInterval: 5 * time.Second,
	}
)

// A timeoutProvider provides values for timeouts to an upgrade controller.
// It is injected into the controller for testability.
type timeoutProvider struct {
	// Do not conclude there were any upgrader errors before this time has elapsed.
	upgraderStartGracePeriod time.Duration

	stuckInSameStateTimeout time.Duration

	stateReconcilerPollInterval time.Duration
}

func (t *timeoutProvider) UpgraderStartGracePeriod() time.Duration {
	return t.upgraderStartGracePeriod
}

func (t *timeoutProvider) StuckInSameStateTimeout() time.Duration {
	return t.stuckInSameStateTimeout
}

func (t *timeoutProvider) StateReconcilePollInterval() time.Duration {
	return t.stateReconcilerPollInterval
}
