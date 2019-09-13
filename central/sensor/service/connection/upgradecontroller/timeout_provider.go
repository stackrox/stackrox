package upgradecontroller

import (
	"time"
)

var (
	defaultTimeoutProvider = timeoutProvider{
		upgraderStartGracePeriod:        time.Minute,
		upgraderStuckInSameStateTimeout: 5 * time.Minute,
		stateReconcilerPollInterval:     5 * time.Second,
		absoluteNoProgressTimeout:       10 * time.Minute,
		rollbackSuccessPeriod:           30 * time.Second,
	}
)

// A timeoutProvider provides values for timeouts to an upgrade controller.
// It is injected into the controller for testability.
type timeoutProvider struct {
	// Do not conclude there were any upgrader errors before this time has elapsed.
	upgraderStartGracePeriod time.Duration

	// Do not worry about sensor's status updates about an upgrader, unless
	// the upgrader has been stuck in the same state for this long.
	upgraderStuckInSameStateTimeout time.Duration

	// Poll at this interval to reconcile the state.
	stateReconcilerPollInterval time.Duration

	// Mark as upgrade as timed out if it's still active, and has been running
	// this long without making progress.
	absoluteNoProgressTimeout time.Duration

	// How long to wait before marking the rollback a success.
	rollbackSuccessPeriod time.Duration
}

func (t *timeoutProvider) UpgraderStartGracePeriod() time.Duration {
	return t.upgraderStartGracePeriod
}

func (t *timeoutProvider) StuckInSameStateTimeout() time.Duration {
	return t.upgraderStuckInSameStateTimeout
}

func (t *timeoutProvider) StateReconcilePollInterval() time.Duration {
	return t.stateReconcilerPollInterval
}

func (t *timeoutProvider) AbsoluteNoProgressTimeout() time.Duration {
	return t.absoluteNoProgressTimeout
}

func (t *timeoutProvider) RollBackSuccessPeriod() time.Duration {
	return t.rollbackSuccessPeriod
}
