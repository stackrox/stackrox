package upgradecontroller

import (
	"context"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
)

// UpgradeController controls auto-upgrading for one specific cluster.
type UpgradeController interface {
	ErrorSignal() concurrency.ReadOnlyErrorSignal
	// RegisterConnection registers a new connection from a sensor, and a handle to send messages to it.
	// Note that callers are responsible for external synchronization -- in particular, they must ensure that
	// a previous call to RegisterConnection has returned before making a new call. Else, the behaviour is undefined.
	RegisterConnection(sensorCtx context.Context, connection common.MessageInjector)
	RecordUpgradeProgress(upgradeProcessID string, upgradeProgress *storage.UpgradeProgress) error
	Trigger(ctx concurrency.Waitable) error
}

type clusterStorage interface {
	UpdateClusterUpgradeStatus(ctx context.Context, clusterID string, status *storage.ClusterUpgradeStatus) error
	GetCluster(ctx context.Context, id string) (*storage.Cluster, bool, error)
}

// New returns a new UpgradeController for the given cluster.
func New(clusterID string, storage clusterStorage) (UpgradeController, error) {
	u := &upgradeController{
		clusterID: clusterID,
		errorSig:  concurrency.NewErrorSignal(),
		storage:   storage,

		upgradeDoneSig: concurrency.NewSignal(),
	}
	err := u.initialize()
	if err != nil {
		return nil, err
	}
	return u, nil
}
