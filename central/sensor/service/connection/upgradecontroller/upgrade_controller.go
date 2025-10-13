package upgradecontroller

import (
	"context"
	"time"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/utils"
)

// SensorConn is the subset of the SensorConnection interface required by the upgrade controller.
type SensorConn interface {
	common.MessageInjector
	CheckAutoUpgradeSupport() error
	SensorVersion() string
}

// UpgradeController controls auto-upgrading for one specific cluster.
type UpgradeController interface {
	// RegisterConnection registers a new connection from a sensor, and a handle to send messages to it.
	// The return value is a once-triggered error waitable that gets triggered if there is any critical issue
	// with the upgrade controller.
	RegisterConnection(sensorCtx context.Context, connection SensorConn) concurrency.ErrorWaitable
	ProcessCheckInFromUpgrader(req *central.UpgradeCheckInFromUpgraderRequest) (*central.UpgradeCheckInFromUpgraderResponse, error)
	ProcessCheckInFromSensor(req *central.UpgradeCheckInFromSensorRequest) error
	Trigger(ctx concurrency.Waitable) error
	TriggerCertRotation(ctx concurrency.Waitable) error
}

// ClusterStorage is the fragment of the cluster store interface that is needed by the upgrade controller.
type ClusterStorage interface {
	UpdateClusterUpgradeStatus(ctx context.Context, clusterID string, status *storage.ClusterUpgradeStatus) error
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
}

// New returns a new UpgradeController for the given cluster.
func New(clusterID string, storage ClusterStorage, autoTriggerEnabledFlag *concurrency.Flag) (UpgradeController, error) {
	return newWithTimeoutProvider(clusterID, storage, autoTriggerEnabledFlag, defaultTimeoutProvider)
}

func validateTimeouts(t timeoutProvider) error {
	errList := errorhelpers.NewErrorList("timeout validation")
	for _, duration := range []time.Duration{t.StuckInSameStateTimeout(), t.UpgraderStartGracePeriod(), t.AbsoluteNoProgressTimeout(), t.StateReconcilePollInterval()} {
		if duration <= 0 {
			errList.AddStringf("invalid duration: %v", duration)
		}
	}
	return errList.ToError()
}

func newWithTimeoutProvider(clusterID string, storage ClusterStorage, autoTriggerEnabledFlag *concurrency.Flag, timeouts timeoutProvider) (UpgradeController, error) {
	if err := validateTimeouts(timeouts); err != nil {
		return nil, utils.ShouldErr(err)
	}

	u := &upgradeController{
		autoTriggerEnabledFlag: autoTriggerEnabledFlag,
		clusterID:              clusterID,
		errorSig:               concurrency.NewErrorSignal(),
		storage:                storage,
		timeouts:               timeouts,
	}

	if err := u.initialize(); err != nil {
		return nil, err
	}
	return u, nil
}
