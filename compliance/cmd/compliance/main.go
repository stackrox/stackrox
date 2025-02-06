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
	cfg := index.DefaultNodeIndexerConfig

	scanner := inventory.NewNodeInventoryComponentScanner(np)
	scanner.Connect(env.NodeScanningEndpoint.Setting())
	cachedNodeIndexer := index.NewCachingNodeIndexer(cfg, env.NodeIndexCacheDuration.DurationSetting(), env.NodeIndexCachePath.Setting())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	umhNodeInv := handler.NewUnconfirmedMessageHandler(ctx, "node-inventory", env.NodeScanningAckDeadlineBase.DurationSetting())
	umhNodeIndex := handler.NewUnconfirmedMessageHandler(ctx, "node-index", env.NodeScanningAckDeadlineBase.DurationSetting())
	c := compliance.NewComplianceApp(np, scanner, cachedNodeIndexer, umhNodeInv, umhNodeIndex)
	c.Start()
}
