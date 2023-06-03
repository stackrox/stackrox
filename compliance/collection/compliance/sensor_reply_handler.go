package compliance

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
)

// SensorReplyHandlerImpl handles ACK/NACK messages from Sensor
type SensorReplyHandlerImpl struct {
	nodeScanner NodeScanner
}

// NewSensorReplyHandlerImpl returns new SensorReplyHandler
func NewSensorReplyHandlerImpl(nodeScanner NodeScanner) *SensorReplyHandlerImpl {
	return &SensorReplyHandlerImpl{
		nodeScanner: nodeScanner,
	}
}

// HandleACK handles ACK message from Sensor
func (s *SensorReplyHandlerImpl) HandleACK(_ context.Context, _ sensor.ComplianceService_CommunicateClient) {
	log.Debugf("Received ACK from Sensor.")
}

// HandleNACK handles NACK message from Sensor
func (s *SensorReplyHandlerImpl) HandleNACK(ctx context.Context, client sensor.ComplianceService_CommunicateClient) {
	go func() {
		log.Infof("Received NACK from Sensor, resending NodeInventory in 10 seconds.")
		select {
		case <-time.After(time.Second * 10):
			s.rescan(ctx, client)
		case <-client.Context().Done():
		case <-ctx.Done():
		}
	}()
}

func (s *SensorReplyHandlerImpl) rescan(ctx context.Context, client sensor.ComplianceService_CommunicateClient) {
	msg, err := s.nodeScanner.ScanNode(ctx)
	if err != nil {
		log.Errorf("Error running ScanNode: %v", err)
		return
	}
	if err := client.Send(msg); err != nil {
		log.Errorf("Error sending to sensor: %v", err)
	}
}
