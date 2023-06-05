package main

import (
	"context"

	"github.com/stackrox/rox/compliance/collection/compliance"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/retry"
)

func main() {
	np := &compliance.EnvNodeNameProvider{}

	scanner := compliance.NewNodeInventoryComponentScanner(np)
	scanner.Connect(env.NodeScanningEndpoint.Setting())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	umh := retry.NewUnconfirmedMessageHandler(ctx, env.NodeScanningAckDeadlineBase.DurationSetting())
	c := compliance.NewComplianceApp(np, scanner, umh)
	c.Start()
}
