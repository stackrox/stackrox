package clusterstatus

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/sensor/common/clusterstatus"
)

type updaterImpl struct {
	updates chan *central.ClusterStatusUpdate

	stopSig concurrency.Signal
}

func (u *updaterImpl) Start() {
	go u.run()
}

func (u *updaterImpl) run() {
	updateMessage := &central.ClusterStatusUpdate{
		Msg: &central.ClusterStatusUpdate_Status{
			Status: &storage.ClusterStatus{
				SensorVersion: version.GetMainVersion(),
			},
		},
	}
	select {
	case u.updates <- updateMessage:
	case <-u.stopSig.Done():
	}
}

func (u *updaterImpl) Stop() {
	u.stopSig.Signal()
}

func (u *updaterImpl) Updates() <-chan *central.ClusterStatusUpdate {
	return u.updates
}

// NewUpdater returns a new ready-to-use updater.
func NewUpdater() clusterstatus.Updater {
	return &updaterImpl{
		updates: make(chan *central.ClusterStatusUpdate),
		stopSig: concurrency.NewSignal(),
	}
}
