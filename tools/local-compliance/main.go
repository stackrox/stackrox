package main

import (
	"context"
	"time"

	compliance "github.com/stackrox/rox/compliance"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry/handler"
)

var log = logging.LoggerForModule()

// local-compliance is an application that allows you to run compliance in your host machine, while using a
// gRPC connection to Sensor. This was introduced for integration-, load-testing, and debugging purposes.
func main() {
	np := &dummyNodeNameProvider{}
	scanner := &LoadGeneratingNodeScanner{
		nodeProvider:       np,
		generationInterval: env.NodeScanningInterval.DurationSetting(),
		initialScanDelay:   env.NodeScanningMaxInitialWait.DurationSetting(),
	}
	nindexer := &LoadGeneratingNodeIndexer{
		generationInterval: env.NodeScanningInterval.DurationSetting(),
		initialScanDelay:   env.NodeScanningMaxInitialWait.DurationSetting(),
	}
	log.Infof("Generation inverval: %v", scanner.generationInterval.String())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	umh1 := handler.NewUnconfirmedMessageHandler(ctx, "node-inventory", 5*time.Second)
	umh2 := handler.NewUnconfirmedMessageHandler(ctx, "node-index", 5*time.Second)
	c := compliance.NewComplianceApp(np, scanner, nindexer, umh1, umh2)
	c.Start()
}

type dummyNodeNameProvider struct{}

func (dnp *dummyNodeNameProvider) GetNodeName() string {
	return "local-compliance"
}
