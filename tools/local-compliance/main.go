package main

import (
	"context"
	"time"

	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var log = logging.LoggerForModule()

// local-compliance is an application that allows you to run compliance in your host machine, while using a
// gRPC connection to Sensor. This was introduced for intergration-, load-testing, and debugging purposes.
func main() {
	np := &dummyNodeNameProvider{}
	scanner := &LoadGeneratingNodeScanner{
		nodeProvider:       np,
		generationInterval: env.NodeScanningInterval.DurationSetting(),
		initialScanDelay:   env.NodeScanningMaxInitialWait.DurationSetting(),
	}
	log.Infof("Generation inverval: %v", scanner.generationInterval.String())

	srh := &dummySensorReplyHandlerImpl{}
	nsr := compliance.NewNodeScanResend(5 * time.Second)
	c := compliance.NewComplianceApp(np, scanner, srh, nsr)
	c.Start()
}

type dummyNodeNameProvider struct{}

func (dnp *dummyNodeNameProvider) GetNodeName() string {
	return "local-compliance"
}

type dummySensorReplyHandlerImpl struct{}

// HandleACK handles ACK message from Sensor
func (s *dummySensorReplyHandlerImpl) HandleACK(_ context.Context, _ sensor.ComplianceService_CommunicateClient) {
	log.Debugf("Received ACK from Sensor.")
}

// HandleNACK handles NACK message from Sensor
func (s *dummySensorReplyHandlerImpl) HandleNACK(_ context.Context, _ sensor.ComplianceService_CommunicateClient) {
	log.Infof("Received NACK from Sensor.")
}
