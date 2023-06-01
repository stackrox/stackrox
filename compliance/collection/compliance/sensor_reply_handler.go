package compliance

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/sensor"
)

type confirmationObserver interface {
	ObserveConfirmation()
}

// SensorReplyHandlerImpl handles ACK/NACK messages from Sensor
type SensorReplyHandlerImpl struct {
	nodeScanner NodeScanner
	scanResend  confirmationObserver
}

// NewSensorReplyHandlerImpl returns new SensorReplyHandler
func NewSensorReplyHandlerImpl(nodeScanner NodeScanner, scanResend confirmationObserver) *SensorReplyHandlerImpl {
	return &SensorReplyHandlerImpl{
		nodeScanner: nodeScanner,
		scanResend:  scanResend,
	}
}

// HandleACK handles ACK message from Sensor
func (s *SensorReplyHandlerImpl) HandleACK(_ context.Context, _ sensor.ComplianceService_CommunicateClient) {
	log.Debugf("Received ACK from Sensor.")
	if s.scanResend != nil {
		s.scanResend.ObserveConfirmation()
	}
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
