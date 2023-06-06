package main

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v3"
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

	back := backoff.NewExponentialBackOff()
	back.InitialInterval = env.NodeScanningAckDeadlineBase.DurationSetting()
	back.RandomizationFactor = 1.0
	back.Multiplier = 1.5
	back.MaxElapsedTime = 30 * time.Minute
	umh := retry.NewUnconfirmedMessageHandler(ctx, back)
	c := compliance.NewComplianceApp(np, scanner, umh)
	c.Start()
}
