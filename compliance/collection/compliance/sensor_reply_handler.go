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
	log.Infof("Received NACK from Sensor, resending NodeInventory in 10 seconds.")
	go func() {
		time.Sleep(time.Second * 10)
		msg, err := s.nodeScanner.ScanNode(ctx)
		if err != nil {
			log.Errorf("error running ScanNode: %v", err)
		} else {
			err := client.Send(msg)
			if err != nil {
				log.Errorf("error sending to sensor: %v", err)
			}
		}
	}()
}
