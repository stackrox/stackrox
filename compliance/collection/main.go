package main

import (
	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/pkg/env"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	np := &compliance.EnvNodeNameProvider{}

	scanner := compliance.NewNodeInventoryComponentScanner(np)
	scanner.Connect(env.NodeScanningEndpoint.Setting())

	srh := compliance.NewSensorReplyHandlerImpl(scanner)
	c := compliance.NewComplianceApp(np, scanner, srh)
	c.Start()
}
