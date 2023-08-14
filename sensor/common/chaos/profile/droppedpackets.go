package profile

import (
	"context"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
)

type droppedPackets struct {
	ctx            context.Context
	percentageLost float32
}

func newDroppedPackets(ctx context.Context, lost float32) *droppedPackets {
	return &droppedPackets{
		ctx:            ctx,
		percentageLost: lost,
	}
}

// Run starts the chaos proxy controller
func (c *droppedPackets) Run(proxy *toxiproxy.Proxy) {
	_, err := proxy.AddToxic("unreliable", "timeout", "", c.percentageLost, toxiproxy.Attributes{
		"timeout": "6000",
	})

	if err != nil {
		log.Warnf("Failed to create toxic for dropping packets: %s", err)
		return
	}
}
