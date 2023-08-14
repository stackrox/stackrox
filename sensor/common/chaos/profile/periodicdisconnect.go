package profile

import (
	"context"
	"time"

	toxiproxy "github.com/Shopify/toxiproxy/v2/client"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type periodicDisconnect struct {
	ctx              context.Context
	disconnectEvery  time.Duration
	disconnectedTime time.Duration
}

func newPeriodicDisconnect(ctx context.Context, every, downtime time.Duration) *periodicDisconnect {
	return &periodicDisconnect{
		ctx:              ctx,
		disconnectEvery:  every,
		disconnectedTime: downtime,
	}
}

// Run starts the chaos proxy controller
func (c *periodicDisconnect) Run(proxy *toxiproxy.Proxy) {
	defer func() {
		// If this function early returns, we want to make sure we stop the goroutine
		// with the proxy enabled (or at least try to).
		_ = proxy.Enable()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-time.After(c.disconnectEvery):
			if err := proxy.Disable(); err != nil {
				log.Warnf("Failed to disable chaos proxy: %s", err)
				return
			}
			log.Debugf("Chaos proxy disabled")
		}

		select {
		case <-c.ctx.Done():
			return
		case <-time.After(c.disconnectedTime):
			if err := proxy.Enable(); err != nil {
				log.Warnf("Failed to re-enable chaos proxy: %s", err)
				return
			}
			log.Debugf("Chaos proxy re-enabled")
		}
	}
}
