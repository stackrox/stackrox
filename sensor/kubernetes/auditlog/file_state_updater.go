package auditlog

import (
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/sync"
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
	auditEventMsgs <-chan *sensor.MsgFromCompliance
	fileStates     map[string]*storage.AuditLogFileState
	fileStateLock  sync.RWMutex
	stopSig        concurrency.Signal
	forceUpdateSig concurrency.Signal
	updateInterval time.Duration
}

func (u *updaterImpl) Start() error {
	go u.runStateSaver()
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

func (u *updaterImpl) runStateSaver() {
	// Can't quit out of this goroutine if the feature is not enabled because it would block the sender of the channel
	// However, if the flag is not enabled, no one should be sending on the channel anyway
	for {
		select {
		case <-u.stopSig.Done():
			return
		case msg := <-u.auditEventMsgs:
			node := msg.Node
			if events := msg.GetAuditEvents(); events != nil && len(events.Events) > 0 {
				// Given how audit logs are always in chronological order, and given how compliance is parsing it in said order,
				// we can make an assumption that the earliest event in this message is still later than the state before
				// But we won't check it, in case there is a corner case where the time is out of order.
				latestTime := events.Events[0].Timestamp
				latestID := events.Events[0].GetId()
				for _, e := range events.Events[1:] {
					if protoutils.After(e.GetTimestamp(), latestTime) {
						latestTime = e.GetTimestamp()
						latestID = e.GetId()
					}
				}
				u.updateFileState(node, latestTime, latestID)
			}
		}
	}
}

func (u *updaterImpl) updateFileState(node string, latestTime *types.Timestamp, latestID string) {
	u.fileStateLock.Lock()
	defer u.fileStateLock.Unlock()

	u.fileStates[node] = &storage.AuditLogFileState{
		CollectLogsSince: latestTime,
		LastAuditId:      latestID,
	}
}

func (u *updaterImpl) runUpdater() {
	if !features.K8sAuditLogDetection.Enabled() {
		log.Info("Stopping audit log file state updater since the flag is disabled")
		return
	}

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
	u.fileStateLock.RLock()
	defer u.fileStateLock.RUnlock()

	// No point in updating if there's no states
	if len(u.fileStates) == 0 {
		return false
	}

	select {
	case u.updates <- u.getUpdateMsgNoLock():
		return false
	case <-u.stopSig.Done():
		return true
	}
}

// fileStateLock must be acquired (at least for read) before calling!
func (u *updaterImpl) getUpdateMsgNoLock() *central.MsgFromSensor {
	nodeStates := make(map[string]*storage.AuditLogFileState, len(u.fileStates))
	for k, v := range u.fileStates {
		nodeStates[k] = v // no need to clone this because when the map is updated a new storage.AuditLogFileState is always created (see updateFileState)
	}

	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_AuditLogStatusInfo{
			AuditLogStatusInfo: &central.AuditLogStatusInfo{
				NodeAuditLogFileStates: nodeStates,
			},
		},
	}
}

// NewUpdater returns an updater that updates central with the latest audit log state for this cluster
// the state is based on the messages received from all compliance nodes in this cluster (in the auditEventMsgs channel)
// updateInterval is optional argument, default 60 seconds interval is used.
func NewUpdater(updateInterval time.Duration, auditEventMsgs <-chan *sensor.MsgFromCompliance) updater.Component {
	interval := updateInterval
	if interval == 0 {
		interval = defaultInterval
	}
	return &updaterImpl{
		updates:        make(chan *central.MsgFromSensor),
		auditEventMsgs: auditEventMsgs,
		fileStates:     make(map[string]*storage.AuditLogFileState),
		stopSig:        concurrency.NewSignal(),
		forceUpdateSig: concurrency.NewSignal(),
		updateInterval: interval,
	}
}
