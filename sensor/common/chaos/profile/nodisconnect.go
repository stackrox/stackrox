package profile

import (
	"context"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
)

type noDisconnect struct {
	ctx context.Context
}

func newNoDisconnect(ctx context.Context) *noDisconnect {
	return &noDisconnect{
		ctx: ctx,
	}
}

// Run starts the chaos proxy controller
func (c *noDisconnect) Run(proxy *toxiproxy.Proxy) {
	if err := proxy.Enable(); err != nil {
		log.Warnf("Failed to enable chaos proxy: %v", err)
		return
	}
	// Wait indefinitely until the context is done.
	<-c.ctx.Done()
}
