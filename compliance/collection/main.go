package main

import (
	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func main() {
	log := logging.LoggerForModule()
	np := &compliance.EnvNodeNameProvider{}

	scanner := compliance.NewNodeInventoryComponentScanner(log, np)
	scanner.Connect(env.NodeScanningEndpoint.Setting())

	srh := compliance.NewSensorReplyHandlerImpl(log, scanner)
	c := compliance.NewComplianceApp(log, np, scanner, srh)
	c.Start()
}
