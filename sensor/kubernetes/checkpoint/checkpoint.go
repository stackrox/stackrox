package checkpoint

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/wal"
)

type checkpointerHandler struct {
	acker wal.MessageAcker
}

func (u *checkpointerHandler) Start() error {
	return nil
}

func (u *checkpointerHandler) Stop(_ error) {}

func (u *checkpointerHandler) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (u *checkpointerHandler) ProcessMessage(msg *central.MsgToSensor) error {
	if checkpointID := msg.GetResourceCheckpoint().GetId(); checkpointID != "" {
		return u.acker.Ack(checkpointID)
	}
	return nil
}

func (u *checkpointerHandler) ResponsesC() <-chan *central.MsgFromSensor {
	return nil
}

func NewCheckpointHandler(acker wal.MessageAcker) common.SensorComponent {
	return &checkpointerHandler{
		acker: acker,
	}
}
