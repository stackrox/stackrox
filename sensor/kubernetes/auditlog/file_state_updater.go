package auditlog

import (
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/compliance"
	"github.com/stackrox/rox/sensor/common/updater"
)

const (
	defaultInterval = 60 * time.Second
)

var (
	log = logging.LoggerForModule()
)

type updaterImpl struct {
	updates        chan *central.MsgFromSensor
	stopSig        concurrency.Signal
	forceUpdateSig concurrency.Signal
	updateInterval time.Duration

	auditLogCollectionManager compliance.AuditLogCollectionManager
}

func (u *updaterImpl) Start() error {
	go u.runUpdater()
	return nil
}

func (u *updaterImpl) Stop(_ error) {
	u.stopSig.Signal()
}

func (u *updaterImpl) Capabilities() []centralsensor.SensorCapability {
	return []centralsensor.SensorCapability{centralsensor.AuditLogEventsCap}
}

func (u *updaterImpl) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (u *updaterImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return u.updates
}

func (u *updaterImpl) ForceUpdate() {
	// If the signal is already triggered then an update will happen soon (or is in process)
	// It will be reset once the update finishes
	u.forceUpdateSig.Signal()
}

func (u *updaterImpl) runUpdater() {
	ticker := time.NewTicker(u.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-u.forceUpdateSig.Done():
			stopped := u.sendUpdate()
			u.forceUpdateSig.Reset()
			if stopped {
				return
			}
		case <-ticker.C:
			if u.sendUpdate() {
				return
			}
		case <-u.stopSig.Done():
			return
		}
	}
}

func (u *updaterImpl) sendUpdate() bool {
	fileStates := u.auditLogCollectionManager.GetLatestFileStates()

	// No point in updating if there's no states
	if len(fileStates) == 0 {
		return false
	}

	select {
	case u.updates <- u.getUpdateMsgNoLock(fileStates):
		return false
	case <-u.stopSig.Done():
		return true
	}
}

func (u *updaterImpl) getUpdateMsgNoLock(fileStates map[string]*storage.AuditLogFileState) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_AuditLogStatusInfo{
			AuditLogStatusInfo: &central.AuditLogStatusInfo{
				NodeAuditLogFileStates: fileStates,
			},
		},
	}
}

// NewUpdater returns an updater that updates central with the latest audit log state for this cluster
// updateInterval is optional argument, default 60 seconds interval is used.
func NewUpdater(updateInterval time.Duration, auditLogCollectionManager compliance.AuditLogCollectionManager) updater.Component {
	interval := updateInterval
	if interval == 0 {
		interval = defaultInterval
	}
	return &updaterImpl{
		updates:                   make(chan *central.MsgFromSensor),
		stopSig:                   concurrency.NewSignal(),
		forceUpdateSig:            concurrency.NewSignal(),
		updateInterval:            interval,
		auditLogCollectionManager: auditLogCollectionManager,
	}
}
