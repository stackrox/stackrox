package main

import (
	"context"
	_ "net/http/pprof" // #nosec G108

	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// local-sensor is an application that allows you to run sensor in your host machine, while mocking a
// gRPC connection to central. This was introduced for testing and debugging purposes. At its current form,
// it does not connect to a real central, but instead it dumps all gRPC messages that would be sent to central in a file.

func main() {
	log := logging.LoggerForModule()
	np := &dummyNodeNameProvider{}
	scanner := &LoadGeneratingNodeScanner{
		log:          log,
		nodeProvider: np,
	}

	srh := &dummySensorReplyHandlerImpl{log: log}
	c := compliance.NewComplianceApp(log, np, scanner, srh)
	c.Start()
}

type dummyNodeNameProvider struct{}

func (dnp *dummyNodeNameProvider) GetNodeName() string {
	return "Foo"
}

type dummySensorReplyHandlerImpl struct {
	log *logging.Logger
}

// HandleACK handles ACK message from Sensor
func (s *dummySensorReplyHandlerImpl) HandleACK(_ context.Context, _ sensor.ComplianceService_CommunicateClient) {
	s.log.Debugf("Received ACK from Sensor.")
}

// HandleNACK handles NACK message from Sensor
func (s *dummySensorReplyHandlerImpl) HandleNACK(_ context.Context, _ sensor.ComplianceService_CommunicateClient) {
	s.log.Infof("Received NACK from Sensor, resending NodeInventory in 10 seconds.")
}
