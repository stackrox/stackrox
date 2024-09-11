package main

import (
	"context"

	"github.com/stackrox/rox/compliance/collection/compliance"
	v4 "github.com/stackrox/rox/compliance/index/v4"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/retry/handler"
)

func init() {
	memlimit.SetMemoryLimit()
}

func main() {
	np := &compliance.EnvNodeNameProvider{}
	cfg := v4.NewNodeIndexerConfigFromEnv()

	scanner := compliance.NewNodeInventoryComponentScanner(np)
	scanner.Connect(env.NodeScanningEndpoint.Setting())
	nodeIndexer := v4.NewNodeIndexer(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	umh := handler.NewUnconfirmedMessageHandler(ctx, env.NodeScanningAckDeadlineBase.DurationSetting())
	c := compliance.NewComplianceApp(np, scanner, nodeIndexer, umh)
	c.Start()
}
