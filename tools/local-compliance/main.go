package main

import (
	"context"
	mathRand "math/rand"
	"time"

	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/generated/internalapi/sensor"
	"github.com/stackrox/rox/pkg/logging"
)

// local-compliance is an application that allows you to run compliance in your host machine, while using a
// gRPC connection to Sensor. This was introduced for intergration-, load-testing, and debugging purposes.
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
	var arr = []string{"foo", "bar", "baz"}
	if len(arr) == 0 {
		return "foo"
	}
	r := mathRand.New(mathRand.NewSource(time.Now().Unix()))
	idx := r.Intn(len(arr))
	return arr[idx]
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
