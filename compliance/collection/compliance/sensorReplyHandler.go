package compliance

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
)

type SensorReplyHandlerImpl struct {
	log         *logging.Logger
	nodeScanner NodeScanner
}

func NewSensorReplyHandlerImpl(log *logging.Logger, nodeScanner NodeScanner) *SensorReplyHandlerImpl {
	return &SensorReplyHandlerImpl{
		log:         log,
		nodeScanner: nodeScanner,
	}
}

func (s *SensorReplyHandlerImpl) HandleACK(ctx context.Context, client sensor.ComplianceService_CommunicateClient) {
	s.log.Debugf("Received ACK from Sensor.")
}

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
