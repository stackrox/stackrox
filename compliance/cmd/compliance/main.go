package main

import (
	"context"

	"github.com/stackrox/rox/compliance"
	"github.com/stackrox/rox/compliance/node"
	"github.com/stackrox/rox/compliance/node/index"
	"github.com/stackrox/rox/compliance/node/inventory"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/memlimit"
	"github.com/stackrox/rox/pkg/retry/handler"
)

func init() {
	memlimit.SetMemoryLimit()
}

func main() {
	np := &node.EnvNodeNameProvider{}
	cfg := index.NewNodeIndexerConfigFromEnv()

	scanner := inventory.NewNodeInventoryComponentScanner(np)
	scanner.Connect(env.NodeScanningEndpoint.Setting())
	cachedNodeIndexer := index.NewCachingNodeIndexer(cfg, env.NodeIndexCacheDuration.DurationSetting(), env.NodeIndexCachePath.Setting())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	umh := handler.NewUnconfirmedMessageHandler(ctx, env.NodeScanningAckDeadlineBase.DurationSetting())
	c := compliance.NewComplianceApp(np, scanner, cachedNodeIndexer, umh)
	c.Start()
}
