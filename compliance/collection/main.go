package main

import (
	"time"

	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/pkg/env"
)

func main() {
	np := &compliance.EnvNodeNameProvider{}

	scanner := compliance.NewNodeInventoryComponentScanner(np)
	scanner.Connect(env.NodeScanningEndpoint.Setting())

	nsr := compliance.NewNodeScanResend(5 * time.Second)
	srh := compliance.NewSensorReplyHandlerImpl(scanner, nsr)
	c := compliance.NewComplianceApp(np, scanner, srh, nsr)
	c.Start()
}
