package main

import (
	"context"
	"time"

	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry/handler"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	umh := handler.NewUnconfirmedMessageHandler(ctx, 5*time.Second)
	c := compliance.NewComplianceApp(np, scanner, umh)
	c.Start()
}

type dummyNodeNameProvider struct{}

func (dnp *dummyNodeNameProvider) GetNodeName() string {
	return "local-compliance"
}
