package compliance

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
)

// SensorReplyHandlerImpl handles ACK/NACK messages from Sensor
type SensorReplyHandlerImpl struct {
	log         *logging.Logger
	nodeScanner NodeScanner
}

// NewSensorReplyHandlerImpl returns new SensorReplyHandler
func NewSensorReplyHandlerImpl(log *logging.Logger, nodeScanner NodeScanner) *SensorReplyHandlerImpl {
	return &SensorReplyHandlerImpl{
		log:         log,
		nodeScanner: nodeScanner,
	}
}

// HandleACK handles ACK message from Sensor
func (s *SensorReplyHandlerImpl) HandleACK(_ context.Context, _ sensor.ComplianceService_CommunicateClient) {
	s.log.Debugf("Received ACK from Sensor.")
}

// HandleNACK handles NACK message from Sensor
func (s *SensorReplyHandlerImpl) HandleNACK(ctx context.Context, client sensor.ComplianceService_CommunicateClient) {
	s.log.Infof("Received NACK from Sensor, resending NodeInventory in 10 seconds.")
	go func() {
		time.Sleep(time.Second * 10)
		msg, err := s.nodeScanner.ScanNode(ctx)
		if err != nil {
			s.log.Errorf("error running ScanNode: %v", err)
		} else {
			err := client.Send(msg)
			if err != nil {
				s.log.Errorf("error sending to sensor: %v", err)
			}
		}
	}()
}
